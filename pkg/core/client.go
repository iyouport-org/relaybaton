package core

/*
Client connection status
 - 0 Opened, Method Request Not Read
 - 1 Method Request accepted
 - 2 Request sent
 - 3 Reply received
 - 4 Close sent
*/

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/iyouport-org/relaybaton/pkg/config"
	"github.com/iyouport-org/relaybaton/pkg/socks5"
	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
	log "github.com/sirupsen/logrus"
	"go.uber.org/fx"
)

type Client struct {
	fx.Lifecycle
	*gnet.EventServer
	httpServer *HTTPServer
	*config.ConfigGo
	*goroutine.Pool
	conns    *Map
	shutdown chan byte
	router   *Router
}

func NewClient(lc fx.Lifecycle, conf *config.ConfigGo, pool *goroutine.Pool, router *Router) *Client {
	client := &Client{
		Lifecycle: lc,
		ConfigGo:  conf,
		Pool:      pool,
		conns:     NewMap(),
		shutdown:  make(chan byte, 10),
		router:    router,
	}

	client.httpServer = &HTTPServer{
		Client: client,
	}

	client.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Debug("start")

			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Debug("stop")
			client.shutdown <- 0x00
			return nil
		},
	})
	return client
}

func (client *Client) Run() error {
	if client.Client.Port != 0 {
		err := gnet.Serve(client, fmt.Sprintf("tcp://:%d", client.Client.Port),
			gnet.WithMulticore(true),
			//gnet.WithReusePort(true),
			gnet.WithLogger(log.StandardLogger()),
			gnet.WithLoadBalancing(gnet.SourceAddrHash),
			gnet.WithTCPKeepAlive(time.Minute))
		if err != nil {
			log.Error(err)
		}
		return err
	}
	return nil
}

func (client *Client) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	select {
	case <-client.shutdown:
		action = gnet.Shutdown
		return
	default:
		key := GetURI(c.RemoteAddr())
		conn, ok := client.conns.Get(key)
		if !ok {
			log.Warn("session do not exist")
			action = gnet.Close
			return
		}
		switch conn.status {
		case StatusOpened:
			mr, err := socks5.NewMethodRequestFrom(frame)
			if err != nil {
				log.Error(err)
				action = gnet.Close
				return
			}
			noAuthRequired := false
			for _, v := range mr.Methods() {
				if v == socks5.MethodNoAuthRequired {
					noAuthRequired = true
					break
				}
			}
			if noAuthRequired {
				mRep := socks5.NewMethodReply(socks5.MethodNoAuthRequired)
				out = mRep.Encode()
			} else {
				log.Error("SOCKS5 error") //TODO
				action = gnet.Close
				return
			}
			conn.status = StatusMethodAccepted
			return out, gnet.None
		case StatusMethodAccepted:
			request, err := socks5.NewRequestFrom(frame)
			if err != nil {
				log.Error(err)
				action = gnet.Close
				return
			}
			conn.dstAddr, err = GetDstAddrFromRequest(request)
			if err != nil {
				log.Error(err)
				action = gnet.Close
				return
			}
			conn.cmd = request.Cmd
			if client.router.Select(conn.dstAddr.(*net.TCPAddr).IP) {
				resp, err := conn.DialWs(request)
				if err != nil {
					log.Error(err)
					action = gnet.Close
					return
				}
				repStr := resp.Get("reply")
				repCode, err := strconv.Atoi(repStr)
				if err != nil {
					log.WithField("rep", repStr).Error(err)
					action = gnet.Close
					return
				}
				reply := socks5.NewReply(byte(repCode), socks5.ATypeIPv4, net.IPv4(127, 0, 0, 1).To4(), client.Client.Port)
				out = reply.Pack()
				err = client.Submit(conn.Run)
				if err != nil {
					log.Error(err)
					action = gnet.Close
					return
				}
				conn.status = StatusAccepted
				return out, gnet.None
			} else { //direct
				conn.tcpConn, err = net.Dial("tcp", conn.dstAddr.String())
				if err != nil {
					log.WithField("Dst Addr", conn.dstAddr.String()).Error(err)
					action = gnet.Close
					return
				}
				reply := socks5.NewReply(socks5.RepSucceeded, socks5.ATypeIPv4, net.IPv4(127, 0, 0, 1).To4(), client.Client.Port)
				out = reply.Pack()
				err = client.Submit(conn.DirectConnect)
				if err != nil {
					log.Error(err)
					action = gnet.Close
					return
				}
				conn.status = StatusAccepted
				return out, gnet.None
			}
		case StatusAccepted:
			var err error
			if client.router.Select(conn.dstAddr.(*net.TCPAddr).IP) {
				err = conn.remoteConn.WriteMessage(websocket.BinaryMessage, frame)
			} else { //direct
				_, err = conn.tcpConn.Write(frame)
			}
			if err != nil {
				log.Error(err)
				action = gnet.Close
			}
			return nil, gnet.None
		default:
			//TODO
		}
		return nil, gnet.None
	}
}

func (client *Client) HandleMethodRequest(data []byte) (b []byte, action gnet.Action) {
	mReq, err := socks5.NewMethodRequestFrom(data)
	if err != nil {
		log.Error(err)
		return nil, gnet.Close
	}
	noAuth := false
	for _, v := range mReq.Methods() {
		if v == socks5.MethodNoAuthRequired {
			noAuth = true
		}
	}
	var mRep socks5.MethodReply
	if noAuth {
		mRep = socks5.NewMethodReply(socks5.MethodNoAuthRequired)
	} else {
		mRep = socks5.NewMethodReply(socks5.MethodNoAuthRequired)
	}
	b = mRep.Encode()
	return b, gnet.None
}

func (client *Client) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	conn := NewConn(c, client.Client)
	client.conns.Put(conn)
	return out, gnet.None
}

func (client *Client) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	if err != nil {
		log.WithFields(log.Fields{
			"localAddr":  c.LocalAddr().String(),
			"remoteAddr": c.RemoteAddr().String(),
		}).Error(err)
	}
	key := GetURI(c.RemoteAddr())
	conn, ok := client.conns.Get(key)
	if ok {
		if conn.remoteConn != nil {
			cErr := conn.remoteConn.Close()
			if cErr != nil {
				log.Error(cErr)
			}
		}
		if conn.tcpConn != nil {
			cErr := conn.tcpConn.Close()
			if cErr != nil {
				log.Error(cErr)
			}
		}
		if conn.dstAddr != nil {
			client.router.RemoveCache(conn.dstAddr.(*net.TCPAddr).IP)
		}
	}
	client.conns.Delete(key)
	return gnet.None
}

func (client *Client) OnInitComplete(svr gnet.Server) (action gnet.Action) {
	go func() {
		if client.Client.HTTPPort != 0 {
			err := client.httpServer.Serve()
			log.Error(err)
		}
	}()
	if client.Client.RedirPort != 0 {
		transparent := RedirServer{
			Client: client,
		}
		go transparent.Run()
	}
	go func() {
		if !client.Client.ProxyAll {
			err := client.router.Update()
			if err != nil {
				log.Error(err)
				return
			}
		}
	}()
	return gnet.None
}

func (client *Client) OnShutdown(svr gnet.Server) {
	log.Debug("shutdown")
	client.Pool.Release()
}
