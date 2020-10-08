package core

import (
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type TCPSegmentConn struct {
	segmentOn bool
	*net.TCPConn
}

func (conn *TCPSegmentConn) Write(b []byte) (n int, err error) {
	if conn.segmentOn {
		packet := gopacket.NewPacket(b, layers.LayerTypeTLS, gopacket.Default)
		if tlsLayer := packet.Layer(layers.LayerTypeTLS); tlsLayer != nil {
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
