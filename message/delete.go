package message

import (
	"encoding/binary"
	"github.com/iyouport-org/relaybaton/util"
)

type DeleteMessage struct {
	Atyp    byte   //1 {0}
	Session uint16 //2
	Length  int
}

func NewDeleteMessage(session uint16) DeleteMessage {
	return DeleteMessage{
		Atyp:    0,
		Session: session,
	}
}

func (dm DeleteMessage) Pack() []byte {
	msgBytes := []byte{0}
	msgBytes = append(msgBytes, util.Uint16ToBytes(dm.Session)...)
	dm.Length = len(msgBytes)
	return msgBytes
}

func UnpackDelete(b []byte) DeleteMessage {
	return DeleteMessage{
		Atyp:    0,
		Session: binary.BigEndian.Uint16(b[1:]),
		Length:  len(b),
	}
}
