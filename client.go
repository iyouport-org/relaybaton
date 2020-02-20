package relaybaton

import (
	"compress/flate"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/iyouport-org/relaybaton/config"
	"github.com/iyouport-org/relaybaton/message"
	"github.com/iyouport-org/relaybaton/util"
	"github.com/iyouport-org/socks5"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/argon2"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client of relaybaton
type Client struct {
	peer
}

// NewClient creates a new client using the given config.
func NewClient(conf config.MainConfig) (*Client, error) {
	var err error
	client := &Client{}
	client.init(conf)

	u := url.URL{
		Scheme: "wss",
		Host:   conf.Client.Server + ":443",
		Path:   "/",
	}

	esniKey, err := getESNIKey(conf.Client.Server)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			ClientESNIKeys: esniKey,
			ServerName:     conf.Client.Server,
		},
		EnableCompression: true,
	}

	header, err := buildHeader(conf)
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

// Run start a client
func (client *Client) Run() {
	sl, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(client.conf.Client.Port))
	if err != nil {
		log.Error(err)
		return
	}
	go client.listenSocks(sl)
	go client.peer.processQueue()

LOOP:
	for {
		select {
		case <-client.close:
			break LOOP
		default:
			client.mutexWsRead.Lock()
			_, content, err := client.wsConn.ReadMessage()
			if err != nil {
				log.Error(err)
				err = client.Close()
				if err != nil {
					log.Error(err)
				}
				break LOOP
			}
			go client.handleWsRead(content)
		}
	}
	err = sl.Close()
	if err != nil {
		log.Error(err)
	}
}

// Dial to the given address from the client and return the connection
func (client *Client) Dial(address string) (net.Conn, error) {
	dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:"+strconv.Itoa(client.conf.Client.Port), nil, nil)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return dialer.Dial("tcp", address)
}

func (client *Client) listenSocks(sl net.Listener) {
	for {
		select {
		case <-client.close:
			return
		default:
			s5conn, err := sl.Accept()
			if err != nil {
				log.Error(err)
				err = client.Close()
				if err != nil {
					log.Error(err)
				}
				return
			}
			go client.serveSocks5(s5conn)
		}
	}
}

func (client *Client) handleWsRead(content []byte) {
	b := make([]byte, len(content))
	copy(b, content)
	client.mutexWsRead.Unlock()
	atyp := b[0]
	session := binary.BigEndian.Uint16(b[1:3])
	if client.connPool.isCloseSent(session) {
		return
	}
	switch atyp {
	case 0: //delete
		msg := message.UnpackDelete(b)
		client.delete(msg.Session)
	case 2: //data
		msg := message.UnpackData(b)
		client.receive(msg)
	case socks5.ATYPIPv4, socks5.ATYPDomain, socks5.ATYPIPv6: //reply {1,3,4}
		msg := message.UnpackReply(b)
		wsw := client.getWebsocketWriter(msg.Session)
		conn := client.connPool.get(msg.Session)
		if conn == nil {
			log.WithField("session", msg.Session).Warn("WebSocket deleted read") //test
			_, err := wsw.writeClose()
			if err != nil {
				log.Error(err)
			}
			return
		}
		err := msg.GetReply().WriteTo(*conn)
		if err != nil {
			log.WithField("session", msg.Session).Warn(err)
			client.connPool.delete(msg.Session)
			_, err = wsw.writeClose()
			if err != nil {
				log.Error(err)
			}
		}
	default: //unknown
		log.WithField("atyp", atyp).Warn("Unknown type message")
	}
}

func (client *Client) resolve(address string) (atyp byte, addr []byte, port uint16, err error) {
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
	}
	return socks5.ATYPIPv6, []byte(addrTCP.IP.To16().String()), uint16(addrTCP.Port), nil
}

func (client *Client) localResolve(request *socks5.Request) (*socks5.Request, error) {
	if request.Atyp == socks5.ATYPDomain && client.conf.DNS.LocalResolve {
		ips, err := net.LookupIP(string(request.DstAddr[1:]))
		//log.WithField("Domain", string(request.BndAddr[1:])).Debug("Looking up IP") //test
		if err != nil {
			log.Debug(err)
			return nil, err
		}
		if len(ips) > 0 {
			for _, ip := range ips {
				if ip.To4() != nil {
					//log.Debug(ip.To4()) //test
					return socks5.NewRequest(request.Cmd, socks5.ATYPIPv4, ip.To4(), request.DstPort), nil
				}
				if ip.To16() != nil {
					//log.Debug(ip.To16()) //test
					return socks5.NewRequest(request.Cmd, socks5.ATYPIPv6, ip.To16(), request.DstPort), nil
				}
			}
		}
	}
	return request, nil
}

func (client *Client) serveSocks5(s5conn net.Conn) {
	port := uint16(s5conn.RemoteAddr().(*net.TCPAddr).Port)
	req, err := client.serveSocks5Negotiation(&s5conn)
	if err != nil {
		log.Error(err)
		err = s5conn.Close()
		if err != nil {
			log.Error(err)
		}
		return
	}
	if (req.Atyp == socks5.ATYPIPv4 || req.Atyp == socks5.ATYPIPv6) && client.conf.Routing.Match(req.DstAddr) {
		rawConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", net.IP(req.DstAddr).String(), binary.BigEndian.Uint16(req.DstPort)))
		if err != nil {
			log.Error(err)
			err = socks5.NewReply(socks5.RepServerFailure, req.Atyp, net.IPv4zero, []byte{0, 0}).WriteTo(s5conn)
			if err != nil {
				log.Error(err)
			}
			return
		}
		err = socks5.NewReply(socks5.RepSuccess, req.Atyp, net.IP{127, 0, 0, 1}, util.Uint16ToBytes(uint16(client.conf.Client.Port))).WriteTo(s5conn)
		if err != nil {
			log.Error(err)
			return
		}
		go SafeCopy(s5conn, rawConn)
		go SafeCopy(rawConn, s5conn)
		return
	}
	wsw := client.getWebsocketWriter(port)
	_, err = wsw.writeConnect(*req)
	if err != nil {
		log.Error(err)
		err = s5conn.Close()
		if err != nil {
			log.Error(err)
		}
		return
	}
	client.connPool.set(port, &s5conn)
	go client.peer.forward(port)
}

func (client *Client) serveSocks5Negotiation(conn *net.Conn) (*socks5.Request, error) {
	negotiationRequest, err := socks5.NewNegotiationRequestFrom(*conn)
	if err != nil {
		log.Error(err)
		return nil, err
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
		return nil, err
	}
	negotiationRely := socks5.NewNegotiationReply(socks5.MethodNone)
	err = negotiationRely.WriteTo(*conn)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	request, err := socks5.NewRequestFrom(*conn)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if request.Cmd != socks5.CmdConnect {
		err = errors.New("command not supported")
		log.Error(err)
		return nil, err
	}
	newRequest, err := client.localResolve(request)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return newRequest, nil
}

func buildHeader(conf config.MainConfig) (http.Header, error) {
	header := http.Header{}
	nonce := make([]byte, 12)
	_, err := io.ReadFull(rand.Reader, nonce)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	key := argon2.Key([]byte(conf.Client.Password), nonce, 3, 32*1024, 4, 32)
	var plaintext = make([]byte, 8)
	binary.BigEndian.PutUint64(plaintext, uint64(time.Now().UnixNano()))

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	cipherText := aesgcm.Seal(nonce, nonce, plaintext, nil)

	header.Add("username", conf.Client.Username)
	header.Add("token", hex.EncodeToString(cipherText))
	return header, nil
}

func getESNIKey(domain string) (*tls.ESNIKeys, error) {
	txt, err := net.LookupTXT("_esni." + domain)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	rawRecord := txt[0]
	esniRecord, err := base64.StdEncoding.DecodeString(rawRecord)
	if err != nil {
		log.WithFields(log.Fields{
			"rawRecord": rawRecord,
		}).Error(err)
		return nil, err
	}
	esniKey, err := tls.ParseESNIKeys(esniRecord)
	if err != nil {
		log.WithFields(log.Fields{
			"esniRecord": esniRecord,
		}).Error(err)
		return nil, err
	}
	return esniKey, nil
}

func SafeCopy(conn1 net.Conn, conn2 net.Conn) {
	_, err := io.Copy(conn1, conn2)
	if err != nil {
		log.Error(err)
		err = conn1.Close()
		if err != nil {
			log.Error(err)
		}
		err = conn2.Close()
		if err != nil {
			log.Error(err)
		}
	}
}
