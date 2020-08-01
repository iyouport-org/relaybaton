package core

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	log "github.com/sirupsen/logrus"
	"net"
)

type TCPSegmentConn struct {
	segmentOn bool
	*net.TCPConn
}

func (conn *TCPSegmentConn) Read(b []byte) (n int, err error) {
	return conn.TCPConn.Read(b)
}

func (conn *TCPSegmentConn) Write(b []byte) (n int, err error) {
	if conn.segmentOn {
		packet := gopacket.NewPacket(b, layers.LayerTypeTLS, gopacket.Default)
		if tlsLayer := packet.Layer(layers.LayerTypeTLS); tlsLayer != nil {
			log.Debug("This is a TLS packet!")
			n, err := conn.TCPConn.Write(b[0:2])
			if err != nil {
				return n, err
			}
			n, err = conn.TCPConn.Write(b[2:])
			if err != nil {
				return n, err
			}
			conn.segmentOn = false
			return len(b), nil
		}
	}
	return conn.TCPConn.Write(b)
}
