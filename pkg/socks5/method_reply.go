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

type MethodReply struct {
	ver byte
	Method
}

func NewMethodReply(method Method) MethodReply {
	return MethodReply{
		ver:    5,
		Method: method,
	}
}

func (mr MethodReply) Encode() []byte {
	return []byte{mr.ver, mr.Method}
}

func (mr MethodReply) WriteTo(writer io.Writer) (int64, error) {
	n, err := writer.Write(mr.Encode())
	return int64(n), err
}
