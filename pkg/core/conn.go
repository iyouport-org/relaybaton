package core

import (
	"compress/flate"
	"context"
	"crypto/sha512"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	relaybatondns "github.com/iyouport-org/relaybaton/pkg/dns"
	"github.com/miekg/dns"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/iyouport-org/relaybaton/pkg/config"
	"github.com/iyouport-org/relaybaton/pkg/socks5"
	"github.com/iyouport-org/relaybaton/pkg/util"
	"github.com/panjf2000/gnet"
	log "github.com/sirupsen/logrus"
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
	tcpConn    net.Conn
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
	httpsrrs, err := relaybatondns.LookupHTTPS(conn.clientConf.Server)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if len(httpsrrs) < 1 {
		err := errors.New("No HTTPS Records")
		log.Error(err)
		return nil, err
	}
	echconfigs, err := getECHConfig(httpsrrs[0])
	if err != nil {
		log.Error(err)
		return nil, err
	}
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			ClientECHConfigs: echconfigs,
			ECHEnabled:       true,
			ServerName:       conn.clientConf.Server,
		},
		NetDial: func(network, addr string) (net.Conn, error) {
			//c, err := net.DialTimeout(network, "1.1.1.1:443", 15*time.Second)
			c, err := net.DialTimeout(network, conn.clientConf.Server+":443", 15*time.Second)
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
			//c, err := net.DialTimeout(network, "1.1.1.1:443", 15*time.Second)
			c, err := net.DialTimeout(network, conn.clientConf.Server+":443", 15*time.Second)
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
			conn.Close()
			return
		}
		if len(b) == 0 {
			continue
		}
		err = conn.localConn.AsyncWrite(b)
		if err != nil {
			log.Error(err)
			conn.Close()
			return
		}
	}
}

func (conn *Conn) DirectConnect() {
	for {
		b := make([]byte, 1<<16)
		n, err := conn.tcpConn.Read(b)
		if err != nil {
			log.Error(err)
			conn.Close()
			return
		}
		err = conn.localConn.AsyncWrite(b[:n])
		if err != nil {
			log.Error(err)
			conn.Close()
			return
		}
	}
}

func (conn *Conn) Close() {
	if conn.localConn != nil {
		err := conn.localConn.Close()
		if err != nil {
			log.Error(err)
		}
	}
	if conn.remoteConn != nil {
		cErr := conn.remoteConn.Close()
		if cErr != nil {
			log.Error(cErr)
		}
	}
	if conn.tcpConn != nil {
		cErr := conn.tcpConn.Close()
		if cErr != nil {
			log.Error(cErr)
		}
	}
}

func (conn *Conn) buildHeader() (http.Header, error) {
	header := http.Header{}
	header.Add("username", conn.clientConf.Username)
	if conn.clientConf.Username == "admin" {
		header.Add("password", conn.clientConf.Password)
	} else {
		sha512key := sha512.Sum512([]byte(conn.clientConf.Password))
		header.Add("password", base64.StdEncoding.EncodeToString(sha512key[:]))
	}
	header.Add("network", "tcp") //TODO
	header.Add("addr", conn.dstAddr.String())
	header.Add("cmd", fmt.Sprintf("%d", conn.cmd))

	return header, nil
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

func getECHConfig(httpsrr dns.HTTPS) ([]tls.ECHConfig, error) {
	for _, v := range httpsrr.Value {
		if v.Key().String() == "echconfig" {
			echconfig, err := base64.StdEncoding.DecodeString(v.String())
			if err != nil {
				log.Error(err)
				return nil, err
			}
			return tls.UnmarshalECHConfigs(echconfig)
		}
	}
	return nil, errors.New("echconfig not found")
}
