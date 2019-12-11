package relaybaton

import (
	"bytes"
	"compress/flate"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/gorilla/websocket"
	"github.com/iyouport-org/socks5"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mssql"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/argon2"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"time"
)

// Server of relaybaton
type Server struct {
	peer
}

// NewServer creates a new server using the given config and websocket connection.
func NewServer(conf Config, wsConn *websocket.Conn) *Server {
	server := &Server{}
	server.init(conf)
	server.wsConn = wsConn
	return server
}

// Run start a server
func (server *Server) Run() {
	go server.peer.processQueue()

	for {
		select {
		case <-server.close:
			return
		default:
			server.mutexWsRead.Lock()
			_, content, err := server.wsConn.ReadMessage()
			if err != nil {
				log.Error(err)
				err = server.Close()
				if err != nil {
					log.Error(err)
				}
				return
			}
			go server.handleWsReadServer(content)
		}
	}
}

func (server *Server) handleWsReadServer(content []byte) {
	b := make([]byte, len(content))
	copy(b, content)
	server.mutexWsRead.Unlock()
	var session uint16
	prefix := binary.BigEndian.Uint16(b[:2])
	switch prefix {
	case 0: //delete
		session = binary.BigEndian.Uint16(b[2:])
		server.delete(session)

	case uint16(socks5.ATYPIPv4), uint16(socks5.ATYPDomain), uint16(socks5.ATYPIPv6):
		session = binary.BigEndian.Uint16(b[2:4])
		dstPort := strconv.Itoa(int(binary.BigEndian.Uint16(b[4:6])))
		ipVer := b[1]
		var dstAddr net.IP
		var err error
		dohProvider := getDoHProvider(server.conf.Server.DoH)
		wsw := server.getWebsocketWriter(session)
		if prefix != uint16(socks5.ATYPDomain) {
			dstAddr = b[6:]
		} else if dohProvider != -1 {
			dstAddr, ipVer, err = nsLookup(bytes.NewBuffer(b[7:]).String(), 6, dohProvider)
		} else {
			var dstAddrs []net.IP
			dstAddrs, err = net.LookupIP(bytes.NewBuffer(b[7:]).String())
			dstAddr = dstAddrs[0]
		}
		if err != nil {
			log.Error(err)
			reply := socks5.NewReply(socks5.RepHostUnreachable, ipVer, net.IPv4zero, []byte{0, 0})
			_, err = wsw.writeReply(*reply)
			if err != nil {
				log.Error(err)
			}
			return
		}
		conn, err := net.Dial("tcp", net.JoinHostPort(dstAddr.String(), dstPort))
		if err != nil {
			log.Error(err)
			reply := socks5.NewReply(socks5.RepServerFailure, ipVer, net.IPv4zero, []byte{0, 0})
			_, err = wsw.writeReply(*reply)
			if err != nil {
				log.Error(err)
			}
			return
		}
		_, addr, port, err := socks5.ParseAddress(conn.LocalAddr().String())
		if err != nil {
			log.Error(err)
			return
		}
		reply := socks5.NewReply(socks5.RepSuccess, ipVer, addr, port)
		_, err = wsw.writeReply(*reply)
		if err != nil {
			log.Error(err)
			return
		}

		server.connPool.set(session, &conn)
		go server.peer.forward(session)

	default:
		session := prefix
		server.receive(session, b[2:])
	}
}

// Handler pass config to ServeHTTP()
type Handler struct {
	Conf Config
	db   *gorm.DB
}

// ServerHTTP accept incoming HTTP request, establish websocket connections, and a new server for handling the connection. If authentication failed, the request will be redirected to the website set in the configuration file.
func (handler Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	upgrader := websocket.Upgrader{
		EnableCompression: true,
	}

	handler.db, err = handler.Conf.DB.getDB()
	if err != nil {
		log.Error(err)
		handler.redirect(&w, r)
		return
	}

	err = handler.authenticate(r.Header)
	if err != nil {
		log.Error(err)
		handler.redirect(&w, r)
		return
	}
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(err)
		handler.redirect(&w, r)
		return
	}
	wsConn.EnableWriteCompression(true)
	err = wsConn.SetCompressionLevel(flate.BestCompression)
	if err != nil {
		log.Error(err)
		return
	}
	server := NewServer(handler.Conf, wsConn)
	go server.Run()
}

func (handler Handler) authenticate(header http.Header) error {
	username := header.Get("username")
	token := header.Get("token")
	data, err := hex.DecodeString(token)
	if err != nil {
		log.Error(err)
		return err
	}
	if len(data) < 12 {
		err = errors.New("authentication failed: data too short")
		log.Error(err)
		return err
	}
	nonce, cipherText := data[:12], data[12:]
	nonceUsed, err := handler.nonceUsed(username, nonce)
	if err != nil {
		log.Error(err)
		return err
	}
	if nonceUsed {
		err = errors.New("authentication failed in nonce verification")
		log.Error(err)
		return err
	}

	password, err := handler.getPassword(username)
	if err != nil {
		log.Error(err)
		return err
	}
	key := argon2.Key([]byte(password), nonce, 3, 32*1024, 4, 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Error(err)
		return err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Error(err)
		return err
	}

	plaintext, err := aesgcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		log.Error(err)
		return err
	}

	t := int64(binary.BigEndian.Uint64(plaintext))
	if time.Since(time.Unix(t/1000000000, t%1000000000)).Seconds() > 60 {
		err = errors.New("authentication failed in time verification")
		log.Error(err)
		return err
	}

	err = handler.saveNonce(username, nonce)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func (handler Handler) getPassword(username string) (string, error) {
	db := handler.db
	db.AutoMigrate(&User{})
	var user User
	err := db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return "", err
	}
	return user.Password, nil
}

func (handler Handler) nonceUsed(username string, nonce []byte) (bool, error) {
	db := handler.db
	db.AutoMigrate(&NonceRecord{})
	var nonceRecord NonceRecord
	db = db.Where(&NonceRecord{Username: username, Nonce: nonce}).First(&nonceRecord)
	if db.RecordNotFound() {
		return false, nil
	}
	err := db.Error
	if err != nil {
		return true, err
	}
	return true, nil
}

func (handler Handler) saveNonce(username string, nonce []byte) error {
	db := handler.db
	db.AutoMigrate(&NonceRecord{})
	newNonce := &NonceRecord{
		Username: username,
		Nonce:    nonce,
	}
	err := db.Create(newNonce).Error
	if err != nil {
		return err
	}
	return nil
}

func (handler Handler) redirect(w *http.ResponseWriter, r *http.Request) {
	newReq, err := http.NewRequest(r.Method, "https://"+handler.Conf.Server.Pretend+r.RequestURI, r.Body)
	if err != nil {
		log.Error(err)
		return
	}
	for k, v := range r.Header {
		newReq.Header.Set(k, v[0])
	}
	resp, err := http.DefaultClient.Do(newReq)
	if err != nil {
		log.Error(err)
		return
	}
	for k, v := range resp.Header {
		(*w).Header().Set(k, v[0])
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return
	}
	err = resp.Body.Close()
	if err != nil {
		log.Error(err)
		return
	}
	_, err = (*w).Write(body)
	if err != nil {
		log.Error(err)
		return
	}
}
