package relaybaton

import (
	"context"
	"encoding/binary"
	"errors"
	"github.com/gorilla/websocket"
	"github.com/iyouport-org/doh-go"
	"github.com/iyouport-org/doh-go/dns"
	"github.com/iyouport-org/socks5"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"sync"
	"time"
)

type peer struct {
	connPool     *connectionPool
	mutexWsRead  sync.Mutex
	controlQueue chan *websocket.PreparedMessage
	messageQueue chan *websocket.PreparedMessage
	hasMessage   chan byte
	close        chan byte
	wsConn       *websocket.Conn
	conf         Config
}

func (peer *peer) init(conf Config) {
	peer.hasMessage = make(chan byte, 2^32+2^16)
	peer.controlQueue = make(chan *websocket.PreparedMessage, 2^16)
	peer.messageQueue = make(chan *websocket.PreparedMessage, 2^32)
	peer.connPool = newConnectionPool()
	peer.close = make(chan byte, 10)
	peer.conf = conf
}

func (peer *peer) forward(session uint16) {
	wsw := peer.getWebsocketWriter(session)
	conn := peer.connPool.get(session)
	_, err := io.Copy(wsw, *conn)
	if err != nil {
		log.Error(err)
	}
	err = (*conn).Close()
	if err != nil {
		log.WithField("session", session).Error(err)
	}
	_, err = wsw.writeClose()
	if err != nil {
		log.WithField("session", session).Error(err)
	}
	peer.connPool.delete(session)
}

func (peer *peer) receive(session uint16, data []byte) {
	wsw := peer.getWebsocketWriter(session)
	conn := peer.connPool.get(session)
	if conn == nil {
		if peer.connPool.isCloseSent(session) {
			return
		}
		log.WithField("session", session).Debug("deleted connection read")
		_, err := wsw.writeClose()
		if err != nil {
			log.Error(err)
		}
		return
	}
	_, err := (*conn).Write(data)
	if err != nil {
		log.WithField("session", session).Error(err)
		err = (*conn).Close()
		if err != nil {
			log.Error(err)
		}
		_, err = wsw.writeClose()
		if err != nil {
			log.Error(err)
		}
		peer.connPool.delete(session)
	}
}

func (peer *peer) delete(session uint16) {
	conn := peer.connPool.get(session)
	if conn != nil {
		err := (*conn).Close()
		if err != nil {
			log.Error(err)
		}
		peer.connPool.delete(session)

		log.Debugf("Port %d Deleted", session)
	}
	peer.connPool.setCloseSent(session)
}

func (peer *peer) getWebsocketWriter(session uint16) webSocketWriter {
	return webSocketWriter{
		session: session,
		peer:    peer,
	}
}

func (peer *peer) processQueue() {
	for {
		select {
		case <-peer.close:
			return
		default:
			<-peer.hasMessage
			if (len(peer.hasMessage)+1)%50 == 0 {
				log.WithField("len", len(peer.hasMessage)+1).Debug("Message Length") //test
			}
			if len(peer.controlQueue) > 0 {
				err := peer.wsConn.WritePreparedMessage(<-peer.controlQueue)
				if err != nil {
					log.Error(err)
					err = peer.Close()
					if err != nil {
						log.Debug(err)
					}
					return
				}
			} else {
				err := peer.wsConn.WritePreparedMessage(<-peer.messageQueue)
				if err != nil {
					log.Error(err)
					err = peer.Close()
					if err != nil {
						log.Debug(err)
					}
					return
				}
			}
		}
	}
}

func (peer *peer) Close() error {
	log.Debug("closing peer")
	err := peer.wsConn.Close()
	if err != nil {
		log.Debug(err)
	}
	peer.close <- 0
	peer.close <- 1
	peer.close <- 2
	return err
}

func uint16ToBytes(n uint16) []byte {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, n)
	return buf
}

func nsLookup(domain string, ipv byte, provider int) (net.IP, byte, error) {
	var dstAddr net.IP
	dstAddr = nil

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c := doh.New(provider)

	if ipv == 6 {
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
	}

	//IPv4
	rsp, err := c.Query(ctx, dns.Domain(domain), dns.TypeA)
	if err != nil {
		log.Error(err)
		return nil, 0, err
	}
	answer := rsp.Answer
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
