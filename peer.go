package relaybaton

import (
	"github.com/gorilla/websocket"
	"github.com/iyouport-org/relaybaton/config"
	"github.com/iyouport-org/relaybaton/message"
	log "github.com/sirupsen/logrus"
	"io"
	"sync"
)

type peer struct {
	connPool     *connectionPool
	mutexWsRead  sync.Mutex
	controlQueue chan *websocket.PreparedMessage
	messageQueue chan *websocket.PreparedMessage
	hasMessage   chan byte
	close        chan byte
	wsConn       *websocket.Conn
	conf         *config.ConfigGo
}

func (peer *peer) init(conf *config.ConfigGo) {
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
		log.WithField("session", session).Trace(err)
	}
	peer.connPool.delete(session)
	_, err = wsw.writeClose()
	if err != nil {
		log.WithField("session", session).Error(err)
	}
}

func (peer *peer) receive(msg message.DataMessage) {
	wsw := peer.getWebsocketWriter(msg.Session)
	conn := peer.connPool.get(msg.Session)
	if conn == nil {
		if peer.connPool.isCloseSent(msg.Session) {
			return
		}
		log.WithField("session", msg.Session).Debug("deleted connection read")
		_, err := wsw.writeClose()
		if err != nil {
			log.Warn(err)
		}
		return
	}
	_, err := (*conn).Write(msg.Data)
	if err != nil {
		log.WithField("session", msg.Session).Debug(err)
		peer.connPool.delete(msg.Session)
		_, err = wsw.writeClose()
		if err != nil {
			log.Warn(err)
		}
	}
}

func (peer *peer) delete(session uint16) {
	conn := peer.connPool.get(session)
	if conn != nil {
		peer.connPool.delete(session)
		log.WithField("session", session).Trace("Port Deleted")
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
			var que *chan *websocket.PreparedMessage
			if len(peer.controlQueue) > 0 {
				que = &peer.controlQueue
			} else {
				que = &peer.messageQueue
			}
			err := peer.wsConn.WritePreparedMessage(<-*que)
			if err != nil {
				log.Error(err)
				err = peer.Close()
				if err != nil {
					log.Warn(err)
				}
				return
			}
		}
	}
}

func (peer *peer) Close() error {
	if len(peer.close) > 0 {
		peer.close <- 0
		return nil
	}
	log.Debug("closing peer")
	peer.close <- 0
	peer.close <- 1
	peer.close <- 2
	err := peer.wsConn.Close()
	if err != nil {
		log.Warn(err)
	}
	peer.close <- 1
	for i := uint16(0); i < 65535; i++ {
		if peer.connPool.get(i) != nil {
			peer.connPool.delete(i)
		}
	}
	return err
}
