package main

import (
	"compress/flate"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/gorilla/websocket"
	"github.com/iyouport-org/doh-go"
	"github.com/iyouport-org/doh-go/dns"
	"github.com/iyouport-org/socks5"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	Peer
}

func NewClient() (*Client, error) {
	client := &Client{}
	client.Init()

	u := url.URL{
		Scheme: "wss",
		Host:   conf.Client.Server + ":443",
		Path:   "/",
	}

	esniKey, err := GetESNIKey(conf.Client.Server)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			ClientESNIKeys: esniKey,
		},
		EnableCompression: true,
	}

	header, err := BuildHeader()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	client.wsConn, _, err = dialer.Dial(u.String(), header)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	client.wsConn.EnableWriteCompression(true)
	err = client.wsConn.SetCompressionLevel(flate.BestCompression)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return client, nil
}

func (client *Client) Run() {
	sl, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(conf.Client.Port))
	if err != nil {
		log.Error(err)
		return
	}
	go client.ListenSocks(sl)
	go client.Peer.ProcessQueue()

	for {
		select {
		case <-client.quit:
			return
		default:
			client.mutexWsRead.Lock()
			_, content, err := client.wsConn.ReadMessage()
			if err != nil {
				log.Error(err)
				client.Close()
				return
			}
			go client.handleWsReadClient(content, client.wsConn)
		}
	}
}

func (client *Client) ListenSocks(sl net.Listener) {
	for {
		select {
		case <-client.quit:
			return
		default:
			s5conn, err := sl.Accept()
			if err != nil {
				log.Error(err)
				client.Close()
				return
			}
			port := uint16(s5conn.RemoteAddr().(*net.TCPAddr).Port)
			wsw := client.GetWebsocketWriter(port)
			err = ServeSocks5(&s5conn, &wsw)
			if err != nil {
				log.Error(err)
				err = s5conn.Close()
				if err != nil {
					log.Error(err)
				}
				continue
			}
			client.connPool.Set(port, &s5conn)
			go client.Peer.Forward(port)
		}
	}
}

func (client *Client) handleWsReadClient(content []byte, wsConn *websocket.Conn) {
	b := make([]byte, len(content))
	copy(b, content)
	client.mutexWsRead.Unlock()
	prefix := binary.BigEndian.Uint16(b[:2])
	if client.connPool.CloseSent(prefix) {
		return
	}
	switch prefix {
	case 0: //delete
		session := binary.BigEndian.Uint16(b[2:])
		client.Delete(session)

	case uint16(socks5.ATYPIPv4), uint16(socks5.ATYPDomain), uint16(socks5.ATYPIPv6):
		atyp := b[1]
		session := binary.BigEndian.Uint16(b[2:4])
		rep := b[4]
		bndPort := b[5:7]
		bndAddr := b[7:]
		reply := socks5.NewReply(rep, atyp, bndAddr, bndPort)
		wsw := client.GetWebsocketWriter(session)
		conn := client.connPool.Get(session)
		if conn == nil {
			log.WithField("session", session).Warnf("WebSocket deleted read") //test
			_, err := wsw.WriteClose()
			if err != nil {
				log.Error(err)
			}
			return
		}
		err := reply.WriteTo(*conn)
		if err != nil {
			log.WithField("session", session).Error(err)
			err = (*conn).Close()
			if err != nil {
				log.Error(err)
			}
			_, err = wsw.WriteClose()
			if err != nil {
				log.Error(err)
			}
			client.connPool.Delete(session)
		}
		if rep != socks5.RepSuccess {
			err = (*client.connPool.Get(session)).Close()
			if err != nil {
				log.Error(err)
			}
			client.connPool.Delete(session)
		}

	default:
		session := prefix
		client.Receive(session, b[2:])
	}
}

func Dial(address string) (net.Conn, error) {
	rawConn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(conf.Client.Port))
	if err != nil {
		log.Error(err)
		return nil, err
	}
	negotiationRequest := socks5.NewNegotiationRequest([]byte{socks5.MethodNone})
	err = negotiationRequest.WriteTo(rawConn)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	negotiationReply, err := socks5.NewNegotiationReplyFrom(rawConn)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if negotiationReply.Method != socks5.MethodNone {
		err = errors.New("unsupported method")
		log.Error(err)
		return nil, err
	}
	atyp, addr, port, err := Resolve(address)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	request := socks5.NewRequest(socks5.CmdConnect, atyp, addr, Uint16ToBytes(port))
	err = request.WriteTo(rawConn)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	reply, err := socks5.NewReplyFrom(rawConn)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if reply.Rep != socks5.RepSuccess {
		err = errors.New("request fail")
		log.WithField("code", reply.Rep).Error(err)
		return nil, err
	}
	return rawConn, nil
}

func Resolve(address string) (atyp byte, addr []byte, port uint16, err error) {
	addrTCP, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		log.Debug(err)
		addrDomain, err := url.Parse("http://" + address)
		if err != nil {
			log.Debug(err)
			return 0, nil, 0, err
		}
		if addrDomain.Port() != "" && addrDomain.Hostname() != "" {
			portInt, err := strconv.Atoi(addrDomain.Port())
			if err != nil {
				log.Debug(err)
				return 0, nil, 0, err
			}
			return socks5.ATYPDomain, []byte(addrDomain.Hostname()), uint16(portInt), nil
		}
		return 0, nil, 0, err
	}
	if addrTCP.IP.To4() != nil {
		return socks5.ATYPIPv4, []byte(addrTCP.IP.To4().String()), uint16(addrTCP.Port), nil
	} else {
		return socks5.ATYPIPv6, []byte(addrTCP.IP.To16().String()), uint16(addrTCP.Port), nil
	}
}

func ServeSocks5(conn *net.Conn, wsw *WebSocketWriter) error {
	negotiationRequest, err := socks5.NewNegotiationRequestFrom(*conn)
	if err != nil {
		log.Error(err)
		return err
	}
	var m byte
	got := false
	for _, m = range negotiationRequest.Methods {
		if m == socks5.MethodNone {
			got = true
		}
	}
	if !got {
		err = errors.New("method not supported")
		log.Error(err)
		return err
	}
	negotiationRely := socks5.NewNegotiationReply(socks5.MethodNone)
	err = negotiationRely.WriteTo(*conn)
	if err != nil {
		log.Error(err)
		return err
	}
	request, err := socks5.NewRequestFrom(*conn)
	if err != nil {
		log.Error(err)
		return err
	}
	if request.Cmd != socks5.CmdConnect {
		err = errors.New("command not supported")
		log.Error(err)
		return err
	}
	_, err = wsw.WriteConnect(*request)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func BuildHeader() (http.Header, error) {
	header := http.Header{}
	h := sha256.New()
	h.Write([]byte(conf.Client.Password))
	key := h.Sum(nil)
	var plaintext = make([]byte, 8)
	binary.BigEndian.PutUint64(plaintext, uint64(time.Now().Unix()))
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	nonce, _ := hex.DecodeString("64a9433eae7ccceee2fc0eda")
	aesGcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	cipherText := aesGcm.Seal(nil, nonce, plaintext, nil)
	header.Add("username", conf.Client.Username)
	header.Add("auth", hex.EncodeToString(cipherText))
	return header, nil
}

func GetESNIKey(domain string) (*tls.ESNIKeys, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c := doh.New(doh.CloudflareProvider)
	rsp, err := c.Query(ctx, dns.Domain("_esni."+domain), dns.TypeTXT)
	if err != nil {
		log.WithField("domain", "_esni."+domain).Error(err)
		return nil, err
	}
	answer := rsp.Answer
	esniRecord, err := base64.StdEncoding.DecodeString(answer[0].Data[1 : len(answer[0].Data)-1])
	if err != nil {
		log.WithField("domain", "_esni."+domain).Error(err)
		return nil, err
	}
	esniKey, err := tls.ParseESNIKeys(esniRecord)
	if err != nil {
		log.WithField("domain", "_esni."+domain).Error(err)
		return nil, err
	}
	return esniKey, nil
}
