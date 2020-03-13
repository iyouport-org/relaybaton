package message

import (
	"encoding/binary"
	"github.com/iyouport-org/relaybaton/util"
)

type DataMessage struct {
	Atyp    byte   //1 {2}
	Session uint16 //2
	Data    []byte //x
	Length  int
}

func NewDataMessage(session uint16, data []byte) DataMessage {
	return DataMessage{
		Atyp:    2,
		Session: session,
		Data:    data,
		Length:  3 + len(data),
	}
}

func (dm DataMessage) Pack() []byte {
	msgBytes := []byte{2}
	msgBytes = append(msgBytes, util.Uint16ToBytes(dm.Session)...)
	msgBytes = append(msgBytes, dm.Data...)
	return msgBytes
}

func UnpackData(b []byte) DataMessage {
	return DataMessage{
		Atyp:    2,
		Session: binary.BigEndian.Uint16(b[1:3]),
		Data:    b[3:],
		Length:  len(b),
	}
}
