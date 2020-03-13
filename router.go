package relaybaton

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/iyouport-org/relaybaton/config"
	"github.com/iyouport-org/relaybaton/util"
	"github.com/iyouport-org/socks5"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"sync"
)

type Router struct {
	conf        *config.ConfigGo
	clients     map[string]*Client
	stats       sync.Map
	mutex       sync.RWMutex
	needRestart chan byte
}

func NewRouter(conf *config.ConfigGo) (router *Router, err error) {
	router = &Router{
		conf:        conf,
		clients:     map[string]*Client{},
		stats:       sync.Map{},
		mutex:       sync.RWMutex{},
		needRestart: make(chan byte, 1),
	}
	router.clients["default"] = nil
	for _, v := range router.conf.Clients.Client {
		client, err := NewClient(router.conf, v)
		if err != nil {
			log.WithField("clients.client.id", v.ID).Error(err)
			return nil, err
		}
		router.clients[v.ID] = client
		router.stats.Store(v.ID, true)
		go router.watchClient(conf, v)
	}
	return router, nil
}

func (router *Router) Run() {
	sl, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", router.conf.Clients.Port))
	if err != nil {
		log.WithField("clients.port", router.conf.Clients.Port).Error(err)
		return
	}
	for {
		s5conn, err := sl.Accept()
		if err != nil {
			log.Error(err)
			return
		}
		go router.serveSocks5(s5conn)
	}
}

func (router *Router) watchClient(conf *config.ConfigGo, confClient *config.ClientGo) {
	for {
		client := router.clients[confClient.ID]
		client.Run()
		router.stats.Store(confClient.ID, false)
		for {
			<-router.ifNeedRestart()
			newClient, err := NewClient(conf, confClient)
			if err != nil {
				log.Warn(err)
				router.setNeedRestart()
				continue
			} else {
				router.clients[confClient.ID] = newClient
				break
			}
		}
		router.stats.Store(confClient.ID, true)
		//router.resetNeedRestart()
	}
}

func (router *Router) serveSocks5(conn net.Conn) {
	port := uint16(conn.RemoteAddr().(*net.TCPAddr).Port)
	request, err := router.serveSocks5Negotiation(conn)
	if err != nil {
		log.Error(err)
		err = conn.Close()
		if err != nil {
			log.Warn(err)
		}
		return
	}
	request, clientKey, err := router.selectClient(request)
	if err != nil {
		log.Error(err)
		err = conn.Close()
		if err != nil {
			log.Warn(err)
		}
		return
	}
	if clientKey == "default" {
		router.directConnect(request, conn)
		return
	}
	if clientKey == "" {
		log.Error("clientKey not found")
		err = conn.Close()
		if err != nil {
			log.Warn(err)
		}
		return
	}
	stat, ok := router.stats.Load(clientKey)
	if !ok {
		log.WithField("client key", clientKey).Error("client not found")
		err = conn.Close()
		if err != nil {
			log.Warn(err)
		}
		return
	}
	if !stat.(bool) {
		router.setNeedRestart()
		err = socks5.NewReply(socks5.RepConnectionRefused, socks5.ATYPIPv4, net.IPv4zero, []byte{0, 0}).WriteTo(conn)
		if err != nil {
			log.Warn(err)
		}
		err = conn.Close()
		if err != nil {
			log.Warn(err)
		}
		return
	}
	client := router.clients[clientKey]
	if client == nil {
		log.WithField("client key", clientKey).Error("client not found")
		err = conn.Close()
		if err != nil {
			log.Warn(err)
		}
		return
	}
	wsw := client.getWebsocketWriter(port, util.GetDstReq(*request))
	_, err = wsw.writeConnect(*request)
	if err != nil {
		log.WithField("session", port).Error(err)
		err = conn.Close()
		if err != nil {
			log.WithField("session", port).Warn(err)
		}
		return
	}
	client.accept(port, util.GetDstReq(*request), &conn)
}

func (router *Router) serveSocks5Negotiation(conn net.Conn) (*socks5.Request, error) { //select client
	negotiationRequest, err := socks5.NewNegotiationRequestFrom(conn)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	got := false
	for _, m := range negotiationRequest.Methods {
		if m == socks5.MethodNone {
			got = true
		}
	}
	if !got {
		err = errors.New(fmt.Sprintf("unknown method: %v", negotiationRequest.Methods))
		log.WithField("methods", negotiationRequest.Methods).Error(err)
		return nil, err
	}
	negotiationRely := socks5.NewNegotiationReply(socks5.MethodNone)
	err = negotiationRely.WriteTo(conn)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	request, err := socks5.NewRequestFrom(conn)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if request.Cmd != socks5.CmdConnect {
		err = errors.New(fmt.Sprintf("unknown command: %d", request.Cmd))
		log.WithField("cmd", request.Cmd).Error(err)
		return nil, err
	}
	return request, nil
}

func (router *Router) selectClient(request *socks5.Request) (_ *socks5.Request, clientKey string, err error) {
	for _, v := range router.conf.Routes.Route {
		switch request.Atyp {
		case socks5.ATYPDomain:
			if v.MatchDomain(string(request.DstAddr[1:])) {
				clientKey = v.Target
				break
			}
		case socks5.ATYPIPv6, socks5.ATYPIPv4:
			if v.MatchIP(request.DstAddr) {
				clientKey = v.Target
				break
			}
		default:
			err = errors.New(fmt.Sprintf("unknown atyp: %d", request.Atyp))
			log.WithField("atyp", request.Atyp).Error(err)
			return nil, clientKey, err
		}
	}
	if router.conf.DNS.LocalResolve && request.Atyp == socks5.ATYPDomain {
		newReq, err := localResolve(request)
		if err != nil {
			log.Warn(err)
			return request, clientKey, err
		}
		request = newReq
		for _, v := range router.conf.Routes.Route {
			if v.MatchIP(request.DstAddr) {
				clientKey = v.Target
				return request, clientKey, nil
			}
		}
	}
	return request, clientKey, nil
}

func (router *Router) directConnect(request *socks5.Request, s5conn net.Conn) {
	var dstAddr string
	port := binary.BigEndian.Uint16(request.DstPort)
	if request.Atyp == socks5.ATYPDomain { //Domain
		dstAddr = string(request.DstAddr[1:])
	} else { //IPv4,IPv6
		dstAddr = net.IP(request.DstAddr).String()
	}
	rawConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", dstAddr, port))
	if err != nil {
		log.WithField("addr", fmt.Sprintf("%s:%d", dstAddr, port)).Error(err)
		err = socks5.NewReply(socks5.RepServerFailure, request.Atyp, net.IPv4zero, []byte{0, 0}).WriteTo(s5conn)
		if err != nil {
			log.WithField("atyp", request.Atyp).Warn(err)
		}
		return
	}
	err = socks5.NewReply(socks5.RepSuccess, request.Atyp, net.IP{127, 0, 0, 1}, util.Uint16ToBytes(router.conf.Clients.Port)).WriteTo(s5conn)
	if err != nil {
		log.WithFields(log.Fields{
			"atyp": request.Atyp,
			"port": router.conf.Clients.Port,
		}).Error(err)
		return
	}
	go SafeCopy(s5conn, rawConn)
	go SafeCopy(rawConn, s5conn)
	return
}

func (router *Router) setNeedRestart() {
	router.mutex.Lock()
	if len(router.needRestart) == 0 {
		router.needRestart <- 1
	}
	router.mutex.Unlock()
}

func (router *Router) ifNeedRestart() chan byte {
	defer router.mutex.RUnlock()
	router.mutex.RLock()
	return router.needRestart
}

func localResolve(request *socks5.Request) (*socks5.Request, error) {
	if request.Atyp == socks5.ATYPDomain {
		ips, err := net.LookupIP(string(request.DstAddr[1:]))
		if err != nil {
			log.WithField("domain", string(request.DstAddr[1:])).Error(err)
			return nil, err
		}
		if len(ips) > 0 {
			for _, ip := range ips {
				if ip.To4() != nil {
					return socks5.NewRequest(request.Cmd, socks5.ATYPIPv4, ip.To4(), request.DstPort), nil
				}
				if ip.To16() != nil {
					return socks5.NewRequest(request.Cmd, socks5.ATYPIPv6, ip.To16(), request.DstPort), nil
				}
			}
		}
	}
	return request, nil
}

func SafeCopy(conn1 net.Conn, conn2 net.Conn) {
	_, err := io.Copy(conn1, conn2)
	if err != nil {
		log.Warn(err)
		err = conn1.Close()
		if err != nil {
			log.Warn(err)
		}
		err = conn2.Close()
		if err != nil {
			log.Warn(err)
		}
	}
}
