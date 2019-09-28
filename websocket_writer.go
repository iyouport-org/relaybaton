package main

import (
	"errors"
	"github.com/gorilla/websocket"
	"github.com/iyouport-org/socks5"
	log "github.com/sirupsen/logrus"
)

type WebSocketWriter struct {
	session uint16
	peer    *Peer
}

func (wsw WebSocketWriter) Write(p []byte) (n int, err error) {
	conn := wsw.peer.connPool.Get(wsw.session)
	if conn == nil {
		err = errors.New("write deleted connection")
		log.WithField("session", wsw.session).Error(err)
		return 0, err
	}
	msg, err := websocket.NewPreparedMessage(websocket.BinaryMessage, append(Uint16ToBytes(wsw.session), p...))
	if err != nil {
		log.Error(err)
	}
	wsw.peer.messageQueue <- msg
	wsw.peer.hasMessage <- byte(1)
	return len(p), err
}

func (wsw WebSocketWriter) WriteClose() (n int, err error) {
	if wsw.peer.connPool.CloseSent(wsw.session) {
		return 0, nil
	} else {
		wsw.peer.connPool.SentClose(wsw.session)
	}
	msg, err := websocket.NewPreparedMessage(websocket.BinaryMessage, append([]byte{0, 0}, Uint16ToBytes(wsw.session)...))
	if err != nil {
		log.Error(err)
	}
	wsw.peer.controlQueue <- msg
	wsw.peer.hasMessage <- byte(1)
	return 2, err
}

func (wsw WebSocketWriter) WriteConnect(request socks5.Request) (n int, err error) {
	// ATYP 2 SESSION 2 PORT 2 ADDR
	msgBytes := []byte{0, request.Atyp}
	msgBytes = append(msgBytes, Uint16ToBytes(wsw.session)...)
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

func (wsw WebSocketWriter) WriteReply(reply socks5.Reply) (n int, err error) {
	// ATYP 2 SESSION 2 REP 1 PORT 2 ADDR
	msgBytes := []byte{0, reply.Atyp}
	msgBytes = append(msgBytes, Uint16ToBytes(wsw.session)...)
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
