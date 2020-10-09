// +build linux
// +build amd64 arm64 riscv64 arm mips mipsle mips64 mips64le ppc64 ppc64le s390x

package core

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"syscall"
	"unsafe"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
)

const SO_ORIGINAL_DST = 80

type RedirServer struct {
	Client   *Client
	listener net.Listener
}

func (server *RedirServer) Run() {
	var err error
	server.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", server.Client.Client.RedirPort))
	if err != nil {
		log.Error(err)
		return
	}
	for {
		leftConn, err := server.listener.Accept()
		if err != nil {
			log.Error(err)
			err = server.listener.Close()
			if err != nil {
				log.Error(err)
			}
			return
		}
		addr, err := realServerAddress(&leftConn)
		if err != nil {
			log.Error(err)
			continue
		}
		dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("localhost:%d", server.Client.Client.Port), nil, nil)
		if err != nil {
			log.Error(err)
			continue
		}
		s5conn, err := dialer.Dial("tcp", addr)
		if err != nil {
			log.Error(err)
			continue
		}
		err = server.Client.Submit(func() {
			io.Copy(s5conn, leftConn)
			s5conn.Close()
			leftConn.Close()
		})
		if err != nil {
			log.Error(err)
			continue
		}
		err = server.Client.Submit(func() {
			io.Copy(leftConn, s5conn)
			s5conn.Close()
			leftConn.Close()
		})
		if err != nil {
			log.Error(err)
			continue
		}
	}
}

type sockaddr struct {
	family uint16
	data   [14]byte
}

//realServerAddress returns an intercepted connection's original destination.
func realServerAddress(conn *net.Conn) (string, error) {
	tcpConn, ok := (*conn).(*net.TCPConn)
	if !ok {
		err := errors.New("not a TCPConn")
		log.Error(err)
		return "", err
	}
	file, err := tcpConn.File()
	if err != nil {
		log.Error(err)
		return "", err
	}
	err = tcpConn.Close()
	if err != nil {
		log.Error()
		return "", err
	}
	*conn, err = net.FileConn(file)
	if err != nil {
		log.Error()
		return "", err
	}
	defer file.Close()
	fd := file.Fd()
	var addr sockaddr
	size := uint32(unsafe.Sizeof(addr))
	err = getsockopt(int(fd), syscall.SOL_IP, SO_ORIGINAL_DST, uintptr(unsafe.Pointer(&addr)), &size)
	if err != nil {
		log.Error(err)
		return "", err
	}
	var ip net.IP
	switch addr.family {
	case syscall.AF_INET:
		ip = addr.data[2:6]
	default:
		err = errors.New("unrecognized address family")
		log.Error(err)
		return "", err
	}
	port := int(addr.data[0])<<8 + int(addr.data[1])
	return net.JoinHostPort(ip.String(), strconv.Itoa(port)), nil
}

func getsockopt(s int, level int, name int, val uintptr, vallen *uint32) (err error) {
	_, _, e1 := syscall.Syscall6(syscall.SYS_GETSOCKOPT, uintptr(s), uintptr(level), uintptr(name), uintptr(val), uintptr(unsafe.Pointer(vallen)), 0)
	if e1 != 0 {
		log.Error(e1)
		err = e1
	}
	return
}
