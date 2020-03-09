package message

import (
	"encoding/binary"
	"github.com/gorilla/websocket"
	"github.com/iyouport-org/relaybaton/util"
	log "github.com/sirupsen/logrus"
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

func (dm DataMessage) Pack2pm() ([]websocket.PreparedMessage, error) {
	n := len(dm.Data) / 1024
	if len(dm.Data)%1024 != 0 {
		n++
	}
	pms := make([]websocket.PreparedMessage, n)
	i := 0
	for i = 0; i < n; i++ {
		msgBytes := []byte{2}
		msgBytes = append(msgBytes, util.Uint16ToBytes(dm.Session)...)
		if i == n-1 {
			msgBytes = append(msgBytes, dm.Data[i*1024:]...)
		} else {
			msgBytes = append(msgBytes, dm.Data[i*1024:(i+1)*1024]...)
		}
		pm, err := websocket.NewPreparedMessage(websocket.BinaryMessage, msgBytes)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		pms[i] = *pm
	}
	return pms, nil
}

func UnpackData(b []byte) DataMessage {
	return DataMessage{
		Atyp:    2,
		Session: binary.BigEndian.Uint16(b[1:3]),
		Data:    b[3:],
		Length:  len(b),
	}
}
