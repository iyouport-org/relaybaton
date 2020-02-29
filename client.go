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
	"time"
)

// Client of relaybaton
type Client struct {
	peer
}

// NewClient creates a new client using the given config.
func NewClient(conf *config.ConfigGo, confClient *config.ClientGo) (*Client, error) {
	log.WithField("id", confClient.ID).Debug("creating new client")
	var err error
	client := &Client{}
	client.init(conf)
	client.timeout = confClient.Timeout

	u := url.URL{
		Scheme: "wss",
		Host:   confClient.Server + ":443",
		Path:   "/",
	}

	var dialer websocket.Dialer
	if confClient.ESNI {
		esniKey, err := getESNIKey(confClient.Server)
		if err != nil {
			log.WithField("server", confClient.Server).Error(err)
			return nil, err
		}
		dialer = websocket.Dialer{
			TLSClientConfig: &tls.Config{
				ClientESNIKeys: esniKey,
				ServerName:     confClient.Server,
			},
			EnableCompression: true,
			HandshakeTimeout:  time.Minute,
		}
	} else {
		dialer = websocket.Dialer{
			EnableCompression: true,
			HandshakeTimeout:  time.Minute,
		}
	}

	header, err := buildHeader(confClient)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	var resp *http.Response
	client.wsConn, resp, err = dialer.Dial(u.String(), header)
	if err != nil {
		fields := log.Fields{}
		if resp != nil {
			fields = util.Header2Fields(resp.Header, resp.Body)
		}
		fields["url"] = u.String()
		log.WithFields(fields).Error(err)
		return nil, err
	}
	client.wsConn.EnableWriteCompression(true)
	err = client.wsConn.SetCompressionLevel(flate.BestCompression)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	err = client.wsConn.SetReadDeadline(time.Now().Add(time.Minute))
	//err = client.wsConn.SetWriteDeadline(time.Now().Add(time.Minute))
	if err != nil {
		log.Error(err)
		return nil, err
	}
	log.WithField("id", confClient.ID).Debug("new client created")
	return client, nil
}

// Run start a client
func (client *Client) Run() {
	go client.processQueue()
	for {
		select {
		case <-client.closing:
			client.closing <- ClientClosed
			return
		default:
			client.mutex.Lock()
			_, content, err := client.wsConn.ReadMessage()
			if err != nil {
				log.Error(err)
				client.mutex.Unlock()
				client.Close()
				return
			}
			err = client.wsConn.SetReadDeadline(time.Now().Add(client.timeout))
			if err != nil {
				log.Error(err)
				client.mutex.Unlock()
				client.Close()
				return
			}
			go client.handleWsRead(content)
		}
	}
}

// Dial to the given address from the client and return the connection
func (client *Client) Dial(address string) (net.Conn, error) {
	dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("127.0.0.1:%d", client.conf.Clients.Port), nil, nil)
	if err != nil {
		log.WithField("addr", address).Error(err)
		return nil, err
	}
	return dialer.Dial("tcp", address)
}

func (client *Client) accept(session uint16, conn *net.Conn) {
	client.connPool.set(session, conn)
	go client.forward(session)
}

func (client *Client) handleWsRead(content []byte) {
	b := make([]byte, len(content))
	copy(b, content)
	client.mutex.Unlock()
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
				log.WithField("session", msg.Session).Trace(err)
			}
			return
		}
		err := msg.GetReply().WriteTo(*conn)
		if err != nil {
			log.WithField("session", msg.Session).Trace(err)
			client.connPool.delete(msg.Session)
			_, err = wsw.writeClose()
			if err != nil {
				log.WithField("session", msg.Session).Warn(err)
			}
		}
	default: //unknown
		log.WithField("atyp", atyp).Warn("Unknown type message")
	}
}

func buildHeader(confClient *config.ClientGo) (http.Header, error) {
	header := http.Header{}
	nonce := make([]byte, 12)
	_, err := io.ReadFull(rand.Reader, nonce)
	if err != nil {
		log.WithField("nonce", nonce).Error(err)
		return nil, err
	}
	key := argon2.Key([]byte(confClient.Password), nonce, 3, 32*1024, 4, 32)
	var plaintext = make([]byte, 8)
	binary.BigEndian.PutUint64(plaintext, uint64(time.Now().UnixNano()))

	block, err := aes.NewCipher(key)
	if err != nil {
		log.WithField("key", key).Error(err)
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		log.WithField("key", key).Error(err)
		return nil, err
	}

	cipherText := aesgcm.Seal(nonce, nonce, plaintext, nil)

	header.Add("username", confClient.Username)
	header.Add("token", hex.EncodeToString(cipherText))
	return header, nil
}

func getESNIKey(domain string) (*tls.ESNIKeys, error) {
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
