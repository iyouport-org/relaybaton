package message

import (
	"encoding/binary"
	"github.com/iyouport-org/relaybaton/util"
	"github.com/iyouport-org/socks5"
)

type ReplyMessage struct {
	Atyp    byte   //1 {1,3,4}
	Session uint16 //2
	Rep     byte   //1
	BndPort uint16 //2
	BndAddr []byte //x
	Length  int
}

func (rm ReplyMessage) Pack() []byte {
	msgBytes := []byte{rm.Atyp}
	msgBytes = append(msgBytes, util.Uint16ToBytes(rm.Session)...)
	msgBytes = append(msgBytes, rm.Rep)
	msgBytes = append(msgBytes, util.Uint16ToBytes(rm.BndPort)...)
	msgBytes = append(msgBytes, rm.BndAddr...)
	rm.Length = len(msgBytes)
	return msgBytes
}

func UnpackReply(b []byte) ReplyMessage {
	return ReplyMessage{
		Atyp:    b[0],
		Session: binary.BigEndian.Uint16(b[1:3]),
		Rep:     b[3],
		BndPort: binary.BigEndian.Uint16(b[4:6]),
		BndAddr: b[6:],
		Length:  len(b),
	}
}

func (rm ReplyMessage) GetReply() *socks5.Reply {
	return socks5.NewReply(rm.Rep, rm.Atyp, rm.BndAddr, util.Uint16ToBytes(rm.BndPort))
}
