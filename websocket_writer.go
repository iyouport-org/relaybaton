package relaybaton

import (
	"errors"
	"github.com/gorilla/websocket"
	"github.com/iyouport-org/socks5"
	log "github.com/sirupsen/logrus"
)

type webSocketWriter struct {
	session uint16
	peer    *peer
}

// Write the given data to the queue waiting to be sent
func (wsw webSocketWriter) Write(p []byte) (n int, err error) {
	conn := wsw.peer.connPool.get(wsw.session)
	if conn == nil {
		err = errors.New("Write deleted connection")
		log.WithField("session", wsw.session).Error(err)
		return 0, err
	}
	msg, err := websocket.NewPreparedMessage(websocket.BinaryMessage, append(uint16ToBytes(wsw.session), p...))
	if err != nil {
		log.Error(err)
	}
	wsw.peer.messageQueue <- msg
	wsw.peer.hasMessage <- byte(1)
	return len(p), err
}

func (wsw webSocketWriter) writeClose() (n int, err error) {
	if wsw.peer.connPool.isCloseSent(wsw.session) {
		return 0, nil
	} else {
		wsw.peer.connPool.setCloseSent(wsw.session)
	}
	msg, err := websocket.NewPreparedMessage(websocket.BinaryMessage, append([]byte{0, 0}, uint16ToBytes(wsw.session)...))
	if err != nil {
		log.Error(err)
	}
	wsw.peer.controlQueue <- msg
	wsw.peer.hasMessage <- byte(1)
	return 2, err
}

func (wsw webSocketWriter) writeConnect(request socks5.Request) (n int, err error) {
	// ATYP 2 SESSION 2 PORT 2 ADDR
	msgBytes := []byte{0, request.Atyp}
	msgBytes = append(msgBytes, uint16ToBytes(wsw.session)...)
	msgBytes = append(msgBytes, request.DstPort...)
	msgBytes = append(msgBytes, request.DstAddr...)
	msg, err := websocket.NewPreparedMessage(websocket.BinaryMessage, msgBytes)
	if err != nil {
		log.Error(err)
	}
	wsw.peer.controlQueue <- msg
	wsw.peer.hasMessage <- byte(1)
	return len(msgBytes), err
}

func (wsw webSocketWriter) writeReply(reply socks5.Reply) (n int, err error) {
	// ATYP 2 SESSION 2 REP 1 PORT 2 ADDR
	msgBytes := []byte{0, reply.Atyp}
	msgBytes = append(msgBytes, uint16ToBytes(wsw.session)...)
	msgBytes = append(msgBytes, reply.Rep)
	msgBytes = append(msgBytes, reply.BndPort...)
	msgBytes = append(msgBytes, reply.BndAddr...)
	msg, err := websocket.NewPreparedMessage(websocket.BinaryMessage, msgBytes)
	if err != nil {
		log.Error(err)
	}
	wsw.peer.controlQueue <- msg
	wsw.peer.hasMessage <- byte(1)
	return len(msgBytes), err
}
