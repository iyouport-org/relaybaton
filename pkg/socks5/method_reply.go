package socks5

import (
	"io"
)

/*
   +----+--------+
   |VER | METHOD |
   +----+--------+
   | 1  |   1    |
   +----+--------+
*/

const (
	MethodNoAcceptable = 0xFF
)

type MethodReply struct {
	ver    byte
	method byte
}

func NewMethodReply(method byte) MethodReply {
	return MethodReply{
		ver:    5,
		method: method,
	}
}

func (mr MethodReply) Encode() []byte {
	return []byte{mr.ver, mr.method}
}

func (mr MethodReply) WriteTo(writer io.Writer) (int64, error) {
	n, err := writer.Write(mr.Encode())
	return int64(n), err
}
