package main

import (
	"bytes"
	"compress/flate"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/gorilla/websocket"
	"github.com/iyouport-org/doh-go"
	"github.com/iyouport-org/doh-go/dns"
	"github.com/iyouport-org/socks5"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"time"
)

type Server struct {
	Peer
}

func NewServer(wsConn *websocket.Conn) *Server {
	server := &Server{}
	server.Init()
	server.wsConn = wsConn

	return server
}

func (server *Server) Run() {
	go server.Peer.ProcessQueue()

	for {
		select {
		case <-server.quit:
			return
		default:
			server.mutexWsRead.Lock()
			_, content, err := server.wsConn.ReadMessage()
			if err != nil {
				log.Error(err)
				server.Close()
				return
			}
			go server.handleWsReadServer(content)
		}
	}
}

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		EnableCompression: true,
	}
	err := Authenticate(r.Header)
	if err != nil {
		log.Error(err)
		Redirect(&w, r)
		return
	}
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(err)
		Redirect(&w, r)
		return
	}
	wsConn.EnableWriteCompression(true)
	err = wsConn.SetCompressionLevel(flate.BestCompression)
	if err != nil {
		log.Error(err)
		return
	}
	server := NewServer(wsConn)
	go server.Run()
}

func (server *Server) handleWsReadServer(content []byte) {
	b := make([]byte, len(content))
	copy(b, content)
	server.mutexWsRead.Unlock()
	var session uint16
	prefix := binary.BigEndian.Uint16(b[:2])
	switch prefix {
	case 0: //delete
		session = binary.BigEndian.Uint16(b[2:])
		server.Delete(session)

	case uint16(socks5.ATYPIPv4), uint16(socks5.ATYPDomain), uint16(socks5.ATYPIPv6):
		session = binary.BigEndian.Uint16(b[2:4])
		dstPort := strconv.Itoa(int(binary.BigEndian.Uint16(b[4:6])))
		ipVer := b[1]
		var dstAddr net.IP
		wsw := server.GetWebsocketWriter(session)
		if prefix != uint16(socks5.ATYPDomain) {
			dstAddr = b[6:]
		} else {
			var err error
			dstAddr, ipVer, err = NsLookup(bytes.NewBuffer(b[7:]).String())
			if err != nil {
				log.Error(err)
				reply := socks5.NewReply(socks5.RepHostUnreachable, ipVer, net.IPv4zero, []byte{0, 0})
				_, err = wsw.WriteReply(*reply)
				if err != nil {
					log.Error(err)
				}
				return
			}
		}
		conn, err := net.Dial("tcp", net.JoinHostPort(dstAddr.String(), dstPort))
		if err != nil {
			log.Error(err)
			reply := socks5.NewReply(socks5.RepServerFailure, ipVer, net.IPv4zero, []byte{0, 0})
			_, err = wsw.WriteReply(*reply)
			if err != nil {
				log.Error(err)
			}
			return
		} else {
			_, addr, port, err := socks5.ParseAddress(conn.LocalAddr().String())
			if err != nil {
				log.Error(err)
				return
			}
			reply := socks5.NewReply(socks5.RepSuccess, ipVer, addr, port)
			_, err = wsw.WriteReply(*reply)
			if err != nil {
				log.Error(err)
				return
			}
		}
		server.connPool.Set(session, &conn)
		go server.Peer.Forward(session)

	default:
		session := prefix
		server.Receive(session, b[2:])
	}
}

func Authenticate(header http.Header) error {
	username := header.Get("username")
	auth := header.Get("auth")
	cipherText, err := hex.DecodeString(auth)
	if err != nil {
		log.Error(err)
		return err
	}
	h := sha256.New()
	h.Write([]byte(GetPassword(username)))
	key := h.Sum(nil)
	nonce, _ := hex.DecodeString("64a9433eae7ccceee2fc0eda")
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Error(err)
		return err
	}
	aesGcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Error(err)
		return err
	}
	plaintext, err := aesGcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		log.Error(err)
		return err
	}
	if time.Since(time.Unix(int64(binary.BigEndian.Uint64(plaintext)), 0)).Seconds() > 60 {
		err = errors.New("authentication fail")
		log.Error(err)
		return err
	}
	return nil
}

func GetPassword(username string) string {
	//TODO
	return conf.Client.Password
}

func NsLookup(domain string) (net.IP, byte, error) {
	var dstAddr net.IP
	dstAddr = nil

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c := doh.New(doh.CloudflareProvider)

	//IPv6
	rsp, err := c.Query(ctx, dns.Domain(domain), dns.TypeAAAA)
	if err != nil {
		log.Error(err)
		return nil, 0, err
	}
	answer := rsp.Answer
	for _, v := range answer {
		if v.Type == 28 {
			dstAddr = net.ParseIP(v.Data).To16()
		}
	}
	if dstAddr != nil {
		return dstAddr, socks5.ATYPIPv6, nil
	}

	//IPv4
	rsp, err = c.Query(ctx, dns.Domain(domain), dns.TypeA)
	if err != nil {
		log.Error(err)
		return nil, 0, err
	}
	answer = rsp.Answer
	for _, v := range answer {
		if v.Type == 1 {
			dstAddr = net.ParseIP(v.Data).To4()
		}
	}
	if dstAddr != nil {
		return dstAddr, socks5.ATYPIPv4, nil
	}

	err = errors.New("DNS error")
	return dstAddr, 0, err
}

func Redirect(w *http.ResponseWriter, r *http.Request) {
	newReq, err := http.NewRequest(r.Method, "https://"+conf.Server.Pretend+r.RequestURI, r.Body)
	if err != nil {
		log.Error(err)
		return
	}
	for k, v := range r.Header {
		newReq.Header.Set(k, v[0])
	}
	resp, err := http.DefaultClient.Do(newReq)
	if err != nil {
		log.Error(err)
		return
	}
	for k, v := range resp.Header {
		(*w).Header().Set(k, v[0])
	}
	body, err := ioutil.ReadAll(resp.Body)
	err = resp.Body.Close()
	if err != nil {
		log.Error(err)
		return
	}
	_, err = (*w).Write(body)
	if err != nil {
		log.Error(err)
		return
	}
}
