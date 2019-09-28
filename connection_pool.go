package main

import (
	"net"
	"sync"
)

type ConnectionPool struct {
	conns     [65535]*net.Conn
	mutex     sync.RWMutex
	closeSent [65535]bool
}

func NewConnectionPool() *ConnectionPool {
	cp := &ConnectionPool{}
	i := 0
	for i < 65535 {
		cp.conns[i] = nil
		i++
	}
	return cp
}

func (cp *ConnectionPool) Get(index uint16) *net.Conn {
	defer cp.mutex.RUnlock()
	cp.mutex.RLock()
	conn := cp.conns[index]
	return conn
}

func (cp *ConnectionPool) Set(index uint16, conn *net.Conn) {
	cp.mutex.Lock()
	cp.conns[index] = conn
	cp.closeSent[index] = false
	cp.mutex.Unlock()
}

func (cp *ConnectionPool) Delete(index uint16) {
	cp.mutex.Lock()
	cp.conns[index] = nil
	cp.mutex.Unlock()
}

func (cp *ConnectionPool) CloseSent(index uint16) bool {
	defer cp.mutex.RUnlock()
	cp.mutex.RLock()
	sent := cp.closeSent[index]
	return sent
}

func (cp *ConnectionPool) SentClose(index uint16) {
	cp.mutex.Lock()
	cp.closeSent[index] = true
	cp.mutex.Unlock()
}
