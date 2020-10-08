package memsocket

import "net"

type Listener struct {
	pipe chan net.Conn
}

// Accept waits for and returns the next connection to the listener.
func (listener *Listener) Accept() (net.Conn, error) {
	return <-listener.pipe, nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (listener *Listener) Close() error {
	return nil
}

// Addr returns the listener's network address.
func (listener *Listener) Addr() net.Addr {
	return Addr{}
}
