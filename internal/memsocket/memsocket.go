package memsocket

import "net"

type MemSocket struct {
	listener *Listener
}

func NewMemSocket() *MemSocket {
	return &MemSocket{listener: &Listener{pipe: make(chan net.Conn, 1024)}}
}

func (memsocket *MemSocket) Dial() net.Conn {
	c1, c2 := net.Pipe()
	memsocket.listener.pipe <- c2
	return c1
}

func (memsocket *MemSocket) Listener() *Listener {
	return memsocket.listener
}
