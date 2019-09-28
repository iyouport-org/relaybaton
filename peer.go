package main

import (
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"io"
	"sync"
)

type Peer struct {
	connPool     *ConnectionPool
	mutexWsRead  sync.Mutex
	controlQueue chan *websocket.PreparedMessage
	messageQueue chan *websocket.PreparedMessage
	hasMessage   chan byte
	quit         chan byte
	wsConn       *websocket.Conn
}

func (peer *Peer) Init() {
	peer.hasMessage = make(chan byte, 2^32+2^16)
	peer.controlQueue = make(chan *websocket.PreparedMessage, 2^16)
	peer.messageQueue = make(chan *websocket.PreparedMessage, 2^32)
	peer.connPool = NewConnectionPool()
	peer.quit = make(chan byte, 3)
}

func (peer *Peer) Forward(session uint16) {
	wsw := peer.GetWebsocketWriter(session)
	conn := peer.connPool.Get(session)
	_, err := io.Copy(wsw, *conn)
	if err != nil {
		log.Error(err)
	}
	err = (*conn).Close()
	if err != nil {
		log.WithField("session", session).Error(err)
	}
	_, err = wsw.WriteClose()
	if err != nil {
		log.WithField("session", session).Error(err)
	}
	peer.connPool.Delete(session)
}

func (peer *Peer) Receive(session uint16, data []byte) {
	wsw := peer.GetWebsocketWriter(session)
	conn := peer.connPool.Get(session)
	if conn == nil {
		if peer.connPool.CloseSent(session) {
			return
		}
		log.WithField("session", session).Debug("deleted connection read")
		_, err := wsw.WriteClose()
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
		_, err = wsw.WriteClose()
		if err != nil {
			log.Error(err)
		}
		peer.connPool.Delete(session)
	}
}

func (peer *Peer) Delete(session uint16) {
	conn := peer.connPool.Get(session)
	if conn != nil {
		err := (*conn).Close()
		if err != nil {
			log.Error(err)
		}
		peer.connPool.Delete(session)

		log.Debugf("Port %d Deleted", session)
	}
	peer.connPool.SentClose(session)
}

func (peer *Peer) GetWebsocketWriter(session uint16) WebSocketWriter {
	return WebSocketWriter{
		session: session,
		peer:    peer,
	}
}

func (peer *Peer) ProcessQueue() {
	for {
		select {
		case <-peer.quit:
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
					peer.Close()
					return
				}
			} else {
				err := peer.wsConn.WritePreparedMessage(<-peer.messageQueue)
				if err != nil {
					log.Error(err)
					peer.Close()
					return
				}
			}
		}
	}
}

func (peer *Peer) Close() {
	log.Debug("closing peer")
	peer.mutexWsRead.Unlock()
	peer.quit <- 0
	peer.quit <- 1
	peer.quit <- 2
}
