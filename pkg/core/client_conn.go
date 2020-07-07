package core

import (
	"crypto/tls"
	"encoding/base64"
	"github.com/dgrr/fastws"
	"time"

	//"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"net"
	"relaybaton/pkg/config"
	//"time"
)

type ClientConn struct {
	*config.ClientGo
	*fastws.Conn
}

func NewClientConn(conf *config.ClientGo) *ClientConn {
	return &ClientConn{
		ClientGo: conf,
	}
}

func (conn *ClientConn) Connect() error {
	log.Debug(conn.Server)
	log.Debug("Fetching ESNI Key")
	esnikey, err := GetESNI(conn.Server)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Debug("Dialing TCP")
	tcpConn, err := net.Dial("tcp", conn.Server+":443")
	//tcpConn, err := net.DialTimeout("tcp", conn.Server+":443", time.Minute)
	if err != nil {
		log.Error(err)
		return err
	}
	err = tcpConn.(*net.TCPConn).SetKeepAlive(true)
	if err != nil {
		log.Error(err)
		return err
	}
	err = tcpConn.(*net.TCPConn).SetKeepAlivePeriod(5 * time.Second)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Debug("Dialing TLS")
	tlsConn := tls.Client(tcpConn, &tls.Config{
		ServerName:     conn.Server,
		ClientESNIKeys: esnikey,
	})
	log.Debug("Dialing Websocket")
	conn.Conn, err = fastws.Client(tlsConn, "wss://"+conn.Server)
	if err != nil {
		log.Error(err)
		return err
	}
	conn.Mode = fastws.ModeBinary
	return nil
}

func GetESNI(domain string) (*tls.ESNIKeys, error) {
	txt, err := net.LookupTXT("_esni." + domain)
	if err != nil {
		log.WithField("domain", domain).Error(err)
		return nil, err
	}
	rawRecord := txt[0]
	esniRecord, err := base64.StdEncoding.DecodeString(rawRecord)
	if err != nil {
		log.WithField("rawRecord", rawRecord).Error(err)
		return nil, err
	}
	esniKey, err := tls.ParseESNIKeys(esniRecord)
	if err != nil {
		log.WithField("esniRecord", esniRecord).Error(err)
		return nil, err
	}
	return esniKey, nil
}
