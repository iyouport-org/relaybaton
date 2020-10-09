// +build !linux linux,386 linux,amd64p32 linux,armbe linux,arm64be linux,mips64p32 linux,mips64p32le linux,ppc linux,riscv linux,s390 linux,sparc linux,sparc64 linux,wasm

package core

import "net"

type RedirServer struct {
	Client   *Client
	listener net.Listener
}

func (server *RedirServer) Run() {

}
