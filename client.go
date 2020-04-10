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
	"github.com/gorilla/websocket"
	"github.com/iyouport-org/relaybaton/config"
	"github.com/iyouport-org/relaybaton/util"
	"github.com/iyouport-org/socks5"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/argon2"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	list       *connList
	confClient *config.ClientGo
}

func NewClient(confClient *config.ClientGo) (*Client, error) {
	return &Client{
		list:       NewConnList(),
		confClient: confClient,
	}, nil
}

func (client *Client) Run(request []byte, outConn net.Conn) {
	conn, err := client.getNewConn()
	if err != nil {
		log.Error(err)
		outConn.Close()
		return
	}
	_, err = conn.Write(request)
	if err != nil {
		log.Error(err)
		outConn.Close()
		conn.Close()
		return
	}
	b, err := conn.ReadMessage()
	if err != nil {
		log.Error(err)
		outConn.Close()
		conn.Close()
		return
	}
	switch b[0] {
	case 0:
		socks5.NewReply(socks5.RepHostUnreachable, socks5.ATYPIPv4, net.IPv4zero, []byte{0, 0}).WriteTo(outConn)
		outConn.Close()
		client.list.Enqueue(conn)
		return
	case 1:
		socks5.NewReply(socks5.RepSuccess, request[0], request[3:], request[1:3]).WriteTo(outConn)
	default:
		//TODO
	}
	if conn.Run(outConn) != nil {
		conn.Close()
		outConn.Close()
	} else {
		client.list.Enqueue(conn)
	}
}

func (client *Client) getNewConn() (*Conn, error) {
	conn := client.list.Dequeue()
	if conn != nil {
		return conn, nil
	} else {
		return client.createNewConn()
	}
}

func (client *Client) createNewConn() (*Conn, error) {
	u := url.URL{
		Scheme: "wss",
		Host:   client.confClient.Server + ":443",
		Path:   "/",
	}
	var dialer websocket.Dialer
	if client.confClient.ESNI {
		esniKey, err := getESNIKey(client.confClient.Server)
		if err != nil {
			log.WithField("server", client.confClient.Server).Error(err)
			return nil, err
		}
		dialer = websocket.Dialer{
			TLSClientConfig: &tls.Config{
				ClientESNIKeys: esniKey,
				ServerName:     client.confClient.Server,
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

	header, err := buildHeader(client.confClient)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	wsConn, resp, err := dialer.Dial(u.String(), header)
	if err != nil {
		fields := log.Fields{}
		if resp != nil {
			fields = util.Header2Fields(resp.Header, resp.Body)
		}
		fields["url"] = u.String()
		log.WithFields(fields).Error(err)
		return nil, err
	}
	wsConn.EnableWriteCompression(true)
	err = wsConn.SetCompressionLevel(flate.BestCompression)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	//log.WithField("id", client.confClient.ID).Debug("new client created")	//test
	return NewConn(wsConn), nil
}

func buildHeader(confClient *config.ClientGo) (http.Header, error) {
	header := http.Header{}
	nonce := make([]byte, 12)
	_, err := io.ReadFull(rand.Reader, nonce)
	if err != nil {
		log.WithField("nonce", nonce).Error(err)
		return nil, err
	}
	key := argon2.Key([]byte(confClient.Password), nonce, 1, 1024, 2, 32)
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
