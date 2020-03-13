package relaybaton

import (
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
)

type connectionPool struct {
	conns     [1 << 16]*net.Conn
	dst       map[uint16]string
	mutex     sync.RWMutex
	closeSent [1 << 16]bool
}

func newConnectionPool() *connectionPool {
	cp := &connectionPool{}
	i := 0
	for i < (1 << 16) {
		cp.conns[i] = nil
		i++
	}
	cp.dst = map[uint16]string{}
	for i = 0; i < (1 << 16); i++ {
		cp.dst[uint16(i)] = ""
	}
	cp.mutex = sync.RWMutex{}
	return cp
}

func (cp *connectionPool) getConn(session uint16) *net.Conn {
	defer cp.mutex.RUnlock()
	cp.mutex.RLock()
	conn := cp.conns[session]
	return conn
}

func (cp *connectionPool) getDst(session uint16) string {
	defer cp.mutex.RUnlock()
	cp.mutex.RLock()
	return cp.dst[session]
}

func (cp *connectionPool) set(session uint16, dst string, conn *net.Conn) {
	cp.mutex.Lock()
	cp.conns[session] = conn
	cp.dst[session] = dst
	cp.closeSent[session] = false
	cp.mutex.Unlock()
}

func (cp *connectionPool) delete(session uint16) {
	cp.mutex.Lock()
	if cp.conns[session] != nil {
		err := (*cp.conns[session]).Close()
		if err != nil {
			log.WithField("session", session).Warn(err)
		}
		cp.conns[session] = nil
		cp.dst[session] = ""
	}
	cp.mutex.Unlock()
}

func (cp *connectionPool) isCloseSent(session uint16) bool {
	defer cp.mutex.RUnlock()
	cp.mutex.RLock()
	sent := cp.closeSent[session]
	return sent
}

func (cp *connectionPool) setCloseSent(session uint16) {
	cp.mutex.Lock()
	cp.closeSent[session] = true
	cp.mutex.Unlock()
}
