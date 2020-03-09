package relaybaton

import (
	"github.com/gorilla/websocket"
	"github.com/iyouport-org/relaybaton/config"
	"github.com/iyouport-org/relaybaton/message"
	log "github.com/sirupsen/logrus"
	"io"
	"sync"
	"time"
)

const (
	PeerClosing = iota
	ForwardClosed
	ProcessQueueClosed
	ClientClosed
	ServerClosed
	PeerClosed
)

type peer struct {
	connPool   *connectionPool
	mutexRead  sync.Mutex
	mutexWrite sync.Mutex
	closing    chan byte
	wsConn     *websocket.Conn
	conf       *config.ConfigGo
	timeout    time.Duration
}

func (peer *peer) init(conf *config.ConfigGo) {
	peer.mutexRead = sync.Mutex{}
	peer.mutexWrite = sync.Mutex{}
	peer.connPool = newConnectionPool()
	peer.closing = make(chan byte, 1<<10)
	peer.conf = conf
}

func (peer *peer) forward(session uint16) {
	wsw := peer.getWebsocketWriter(session)
	conn := peer.connPool.get(session)
	if conn == nil {
		log.WithField("session", session).Warn("trying forward nil conn")
		return
	}
	_, err := peer.copy(wsw, *conn)
	if err != nil {
		log.WithField("session", session).Trace(err)
		peer.connPool.delete(session)
		if err.Error() == "peer closing" {
			return
		}
		_, err = wsw.writeClose()
		if err != nil {
			log.WithField("session", session).Error(err)
		}
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
	}
	peer.connPool.setCloseSent(session)
}

func (peer *peer) getWebsocketWriter(session uint16) webSocketWriter {
	return webSocketWriter{
		session: session,
		peer:    peer,
	}
}

func (peer *peer) copy(dst io.Writer, src io.Reader) (written int64, err error) {
	size := 32 * 1024
	if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
		if l.N < 1 {
			size = 1
		} else {
			size = int(l.N)
		}
	}
	buf := make([]byte, size)
LOOP:
	for {
		select {
		case <-peer.closing:
			peer.closing <- ForwardClosed
			return written, err
		default:
			nr, er := src.Read(buf)
			if nr > 0 {
				nw, ew := dst.Write(buf[0:nr])
				if nw > 0 {
					written += int64(nw)
				}
				if ew != nil {
					err = ew
					break LOOP
				}
				if nr != nw {
					err = io.ErrShortWrite
					break LOOP
				}
			}
			if er != nil {
				if er != io.EOF {
					err = er
				}
				break LOOP
			}
		}
	}
	return written, err
}

func (peer *peer) Close() {
	if len(peer.closing) > 0 {
		peer.closing <- PeerClosing
		return
	}
	log.Debug("closing peer")
	peer.closing <- PeerClosing
	peer.closing <- PeerClosing
	peer.closing <- PeerClosing

	err := peer.wsConn.Close()
	if err != nil {
		log.Warn(err)
	}

	for i := 0; i < 1<<16; i++ {
		if peer.connPool.get(uint16(i)) != nil {
			peer.connPool.delete(uint16(i))
		}
	}
	peer.closing <- PeerClosed
	log.Debug("peer closed")
	return
}
