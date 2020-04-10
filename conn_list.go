package relaybaton

import (
	"container/list"
	"sync"
)

type connList struct {
	list  *list.List
	mutex sync.Mutex
}

func NewConnList() *connList {
	return &connList{
		list: list.New(),
	}
}

func (list *connList) Enqueue(conn *Conn) {
	list.mutex.Lock()
	conn.End()
	list.list.PushBack(conn)
	list.mutex.Unlock()
}

func (list *connList) Dequeue() *Conn {
	list.mutex.Lock()
	defer list.mutex.Unlock()
	v := list.list.Front()
	if v != nil {
		conn := v.Value.(*Conn)
		list.list.Remove(v)
		return conn
	} else {
		return nil
	}
}
