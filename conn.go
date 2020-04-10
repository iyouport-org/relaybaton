package relaybaton

import (
	"bytes"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

type Conn struct {
	rawConn    *websocket.Conn
	outConn    net.Conn
	mutexRead  sync.Mutex
	mutexWrite sync.Mutex
	closing    chan byte
	closed     sync.WaitGroup
}

func NewConn(wsConn *websocket.Conn) *Conn {
	return &Conn{
		rawConn: wsConn,
		outConn: nil,
		closing: make(chan byte, 4),
	}
}

func (conn *Conn) Run(outConn net.Conn) net.Conn {
	conn.outConn = outConn
	err := conn.SetDeadline(time.Time{})
	if err != nil {
		log.Error(err)
	}
	conn.closed.Add(2)
	conn.closing = make(chan byte, 4)
	go conn.Forward()
	go conn.Copy()
	conn.closed.Wait()
	return conn.outConn
}

// Read reads data from the connection.
// Read can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (conn *Conn) Read(b []byte) (n int, err error) {
	buffer := bytes.NewBuffer(b)
	conn.mutexRead.Lock()
	_, p, err := conn.rawConn.ReadMessage()
	conn.mutexRead.Unlock()
	if err != nil {
		return 0, err
	} else {
		buffer.Write(p)
		return len(b), nil
	}
}

func (conn *Conn) ReadMessage() (p []byte, err error) {
	conn.mutexRead.Lock()
	_, p, err = conn.rawConn.ReadMessage()
	conn.mutexRead.Unlock()
	return p, err
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (conn *Conn) Write(b []byte) (n int, err error) {
	n = len(b)
	pm, err := websocket.NewPreparedMessage(websocket.BinaryMessage, b)
	if err != nil {
		log.Error(err)
		return 0, err
	}
	conn.mutexWrite.Lock()
	err = conn.rawConn.WritePreparedMessage(pm)
	conn.mutexWrite.Unlock()
	if err != nil {
		n = 0
	}
	return n, err
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (conn *Conn) Close() error {
	conn.closing <- 0
	conn.closing <- 0
	conn.closed.Wait()
	return conn.rawConn.Close()
}

func (conn *Conn) End() {
	conn.closing <- 0
	conn.closing <- 0
	if conn.outConn != nil {
		err := conn.outConn.Close()
		if err != nil {
			log.Error(err)
		}
	}
	conn.closed.Wait()
	conn.outConn = nil
	err := conn.SetDeadline(time.Now().Add(time.Minute))
	if err != nil {
		log.Error(err)
	}
}

// LocalAddr returns the local network address.
func (conn *Conn) LocalAddr() net.Addr {
	return conn.rawConn.UnderlyingConn().LocalAddr()
}

// RemoteAddr returns the remote network address.
func (conn *Conn) RemoteAddr() net.Addr {
	return conn.rawConn.UnderlyingConn().RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail with a timeout (see type Error) instead of
// blocking. The deadline applies to all future and pending
// I/O, not just the immediately following call to Read or
// Write. After a deadline has been exceeded, the connection
// can be refreshed by setting a deadline in the future.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
//
// Note that if a TCP connection has keep-alive turned on,
// which is the default unless overridden by Dialer.KeepAlive
// or ListenConfig.KeepAlive, then a keep-alive failure may
// also return a timeout error. On Unix systems a keep-alive
// failure on I/O can be detected using
// errors.Is(err, syscall.ETIMEDOUT).
func (conn *Conn) SetDeadline(t time.Time) error {
	err := conn.rawConn.SetReadDeadline(t)
	if err != nil {
		return err
	}
	return conn.rawConn.SetWriteDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (conn *Conn) SetReadDeadline(t time.Time) error {
	return conn.rawConn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (conn *Conn) SetWriteDeadline(t time.Time) error {
	return conn.rawConn.SetWriteDeadline(t)
}

func (conn *Conn) Forward() {
	defer conn.closed.Done()
	for {
		select {
		case <-conn.closing:
			return
		default:
			p, err := conn.ReadMessage()
			if err != nil {
				log.Error(err)
				go conn.Close()
				<-conn.closing
				return
			}
			_, err = conn.outConn.Write(p)
			if err != nil {
				log.Error(err)
				go conn.End()
				<-conn.closing
				return
			}
		}
	}
}

func (conn *Conn) Copy() {
	defer conn.closed.Done()
	size := 32 * 1024
	buf := make([]byte, size)
	for {
		select {
		case <-conn.closing:
			return
		default:
			nr, er := conn.outConn.Read(buf)
			if er != nil {
				log.Error(er)
				go conn.End()
				<-conn.closing
				return
			}
			if nr > 0 {
				_, ew := conn.Write(buf[0:nr])
				if ew != nil {
					log.Error(ew)
					go conn.Close()
					<-conn.closing
					return
				}
			}
		}
	}
}
