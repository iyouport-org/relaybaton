// +build !linux

package core

import "net"

type TransparentServer struct {
	Client   *Client
	listener net.Listener
}

func (server *TransparentServer) Run() {

}
