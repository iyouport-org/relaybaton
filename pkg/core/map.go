package core

import (
	"fmt"
	"net"
	"sync"

	"github.com/emirpasic/gods/maps/hashmap"
)

type Map struct {
	mutex   sync.RWMutex
	connMap *hashmap.Map
}

func NewMap() *Map {
	return &Map{
		mutex:   sync.RWMutex{},
		connMap: hashmap.New(),
	}
}

func (m *Map) Put(conn *Conn) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.connMap.Put(conn.key, conn)
}

func (m *Map) Get(key string) (*Conn, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	v, ok := m.connMap.Get(key)
	return v.(*Conn), ok
}

func (m *Map) Delete(key string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.connMap.Remove(key)
}

func GetURI(addr net.Addr) string {
	return fmt.Sprintf("%s://%s", addr.Network(), addr.String())
}
