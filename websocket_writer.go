package relaybaton

import (
	"encoding/binary"
	"errors"
	"github.com/gorilla/websocket"
	"github.com/iyouport-org/relaybaton/message"
	"github.com/iyouport-org/socks5"
	log "github.com/sirupsen/logrus"
)

type webSocketWriter struct {
	session uint16
	peer    *peer
}

// Write the given data to the queue waiting to be sent
func (wsw webSocketWriter) Write(b []byte) (n int, err error) {
	conn := wsw.peer.connPool.get(wsw.session)
	if conn == nil {
		err = errors.New("write deleted connection")
		log.WithField("session", wsw.session).Error(err)
		return 0, err
	}
	msg := message.NewDataMessage(wsw.session, b)
	pMsg, err := websocket.NewPreparedMessage(websocket.BinaryMessage, msg.Pack())
	if err != nil {
		log.WithField("msg", msg.Pack()).Error(err)
		return 0, err
	}
	wsw.peer.mutexWrite.Lock()
	err = wsw.peer.wsConn.WritePreparedMessage(pMsg)
	if err != nil {
		log.WithField("session", wsw.session).Error(err)
		wsw.peer.Close()
		wsw.peer.mutexWrite.Unlock()
		return 0, err
	}
	wsw.peer.mutexWrite.Unlock()
	return len(b), err
}

func (wsw webSocketWriter) writeClose() (n int, err error) {
	if wsw.peer.connPool.isCloseSent(wsw.session) {
		return 0, nil
	}
	wsw.peer.connPool.setCloseSent(wsw.session)
	msg := message.NewDeleteMessage(wsw.session)
	pMsg, err := websocket.NewPreparedMessage(websocket.BinaryMessage, msg.Pack())
	if err != nil {
		log.WithField("msg", msg.Pack()).Error(err)
		return 0, err
	}
	wsw.peer.mutexWrite.Lock()
	err = wsw.peer.wsConn.WritePreparedMessage(pMsg)
	if err != nil {
		log.WithField("session", wsw.session).Error(err)
		wsw.peer.Close()
		wsw.peer.mutexWrite.Unlock()
		return 0, err
	}
	wsw.peer.mutexWrite.Unlock()
	return 2, err
}

func (wsw webSocketWriter) writeConnect(request socks5.Request) (n int, err error) {
	msg := message.ConnectMessage{
		Atyp:    request.Atyp,
		Session: wsw.session,
		DstPort: binary.BigEndian.Uint16(request.DstPort),
		DstAddr: request.DstAddr,
	}
	pMsg, err := websocket.NewPreparedMessage(websocket.BinaryMessage, msg.Pack())
	if err != nil {
		log.WithField("msg", msg.Pack()).Error(err)
		return 0, err
	}
	wsw.peer.mutexWrite.Lock()
	err = wsw.peer.wsConn.WritePreparedMessage(pMsg)
	if err != nil {
		log.WithField("session", wsw.session).Error(err)
		wsw.peer.Close()
		wsw.peer.mutexWrite.Unlock()
		return 0, err
	}
	wsw.peer.mutexWrite.Unlock()
	return msg.Length, err
}

func (wsw webSocketWriter) writeReply(reply socks5.Reply) (n int, err error) {
	msg := message.ReplyMessage{
		Atyp:    reply.Atyp,
		Session: wsw.session,
		Rep:     reply.Rsv,
		BndPort: binary.BigEndian.Uint16(reply.BndPort),
		BndAddr: reply.BndAddr,
	}
	pMsg, err := websocket.NewPreparedMessage(websocket.BinaryMessage, msg.Pack())
	if err != nil {
		log.WithField("msg", msg.Pack()).Error(err)
		return 0, err
	}
	//wsw.peer.controlQueue <- pMsg
	//wsw.peer.hasMessage <- byte(1)
	wsw.peer.mutexWrite.Lock()
	err = wsw.peer.wsConn.WritePreparedMessage(pMsg)
	if err != nil {
		log.WithField("session", wsw.session).Error(err)
		wsw.peer.Close()
		wsw.peer.mutexWrite.Unlock()
		return 0, err
	}
	wsw.peer.mutexWrite.Unlock()
	return msg.Length, err
}
