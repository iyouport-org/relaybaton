package core

import (
	"compress/flate"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/fasthttp/websocket"
	"github.com/panjf2000/gnet"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"net/url"
	"relaybaton/pkg/config"
	"relaybaton/pkg/socks5"
	"relaybaton/pkg/util"
	"time"
)

const (
	StatusOpened         = uint8(0x0)
	StatusMethodAccepted = uint8(0x1)
	StatusAccepted       = uint8(0x2)
)

type Conn struct {
	key        string
	status     uint8
	dstAddr    net.Addr
	cmd        socks5.Cmd
	localConn  gnet.Conn
	remoteConn *websocket.Conn
	clientConf *config.ClientGo
}

func NewConn(gnetConn gnet.Conn, clientConf *config.ClientGo) *Conn {
	return &Conn{
		key:        GetURI(gnetConn.RemoteAddr()),
		status:     StatusOpened,
		localConn:  gnetConn,
		remoteConn: nil,
		clientConf: clientConf,
	}
}

func (conn *Conn) DialWs(request socks5.Request) (http.Header, error) {
	u := url.URL{
		Scheme: "wss",
		Host:   conn.clientConf.Server + ":443",
		Path:   "/",
	}
	esnikey, err := GetESNI(conn.clientConf.Server)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			ClientESNIKeys: esnikey,
			ServerName:     conn.clientConf.Server,
		},
		NetDial: func(network, addr string) (net.Conn, error) {
			c, err := net.DialTimeout(network, "1.1.1.1:443", 15*time.Second)
			if err != nil {
				return nil, err
			}
			err = c.(*net.TCPConn).SetKeepAlive(true)
			if err != nil {
				return nil, err
			}
			return &TCPSegmentConn{
				segmentOn: true,
				TCPConn:   c.(*net.TCPConn),
			}, nil
		},
		NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			c, err := net.DialTimeout(network, "1.1.1.1:443", 15*time.Second)
			if err != nil {
				return nil, err
			}
			err = c.(*net.TCPConn).SetKeepAlive(true)
			if err != nil {
				return nil, err
			}
			return &TCPSegmentConn{
				segmentOn: true,
				TCPConn:   c.(*net.TCPConn),
			}, nil
		},
		EnableCompression: true,
		HandshakeTimeout:  time.Minute,
	}
	header, err := conn.buildHeader()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	var resp *http.Response
	conn.remoteConn, resp, err = dialer.Dial(u.String(), header)
	if err != nil {
		fields := log.Fields{}
		if resp != nil {
			fields = util.Header2Fields(resp.Header, resp.Body)
		}
		fields["url"] = u.String()
		log.WithFields(fields).Error(err)
		return nil, err
	}
	err = conn.remoteConn.SetCompressionLevel(flate.BestCompression)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return resp.Header, err
}

func (conn *Conn) Run() {
	for {
		_, b, err := conn.remoteConn.ReadMessage()
		if err != nil {
			log.Error(err)
			conn.remoteConn.Close()
			return
		}
		if len(b) == 0 {
			continue
		}
		err = conn.localConn.AsyncWrite(b)
		if err != nil {
			log.Error(err)
			conn.remoteConn.Close()
			return
		}
	}
}

func (conn *Conn) buildHeader() (http.Header, error) {
	header := http.Header{}
	header.Add("username", conn.clientConf.Username)
	header.Add("password", conn.clientConf.Password)
	header.Add("network", "tcp") //TODO
	header.Add("addr", conn.dstAddr.String())
	header.Add("cmd", fmt.Sprintf("%d", conn.cmd))

	return header, nil
}

func GetESNI(domain string) (*tls.ESNIKeys, error) {
	txt, err := net.DefaultResolver.LookupTXT(context.Background(), "_esni."+domain)
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

func GetDstAddrFromRequest(request socks5.Request) (net.Addr, error) {
	switch request.ATyp {
	case socks5.ATypeIPv4:
		return &net.TCPAddr{
			IP:   net.IP(request.DstAddr).To4(),
			Port: int(request.DstPort),
		}, nil
	case socks5.ATypeIPv6:
		return &net.TCPAddr{
			IP:   net.IP(request.DstAddr).To16(),
			Port: int(request.DstPort),
		}, nil
	case socks5.ATypeDomainName:
		ipAddrs, err := net.DefaultResolver.LookupIPAddr(context.Background(), string(request.DstAddr[1:]))
		if err != nil {
			log.Error(err)
			return nil, err
		}
		if len(ipAddrs) > 0 {
			return &net.TCPAddr{
				IP:   ipAddrs[0].IP,
				Port: int(request.DstPort),
				Zone: ipAddrs[0].Zone,
			}, nil
		} else {
			log.Error(err)
			return nil, err
		}
	default:
		err := errors.New("unknown aTyp")
		log.WithField("aTyp", request.ATyp).Error(err)
		return nil, err
	}
}
