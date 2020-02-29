package relaybaton

import (
	"compress/flate"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/iyouport-org/relaybaton/config"
	"github.com/iyouport-org/relaybaton/message"
	"github.com/iyouport-org/relaybaton/util"
	"github.com/iyouport-org/socks5"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mssql"    //mssql
	_ "github.com/jinzhu/gorm/dialects/mysql"    //mysql
	_ "github.com/jinzhu/gorm/dialects/postgres" //postgres
	_ "github.com/jinzhu/gorm/dialects/sqlite"   //sqlite
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/argon2"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

// Server of relaybaton
type Server struct {
	peer
}

// NewServer creates a new server using the given config and websocket connection.
func NewServer(conf *config.ConfigGo, wsConn *websocket.Conn) *Server {
	server := &Server{}
	server.init(conf)
	server.timeout = conf.Server.Timeout
	server.wsConn = wsConn
	return server
}

// Run start a server
func (server *Server) Run() {
	go server.processQueue()

	for {
		select {
		case <-server.closing:
			server.closing <- ServerClosed
			return
		default:
			server.mutex.Lock()
			_, content, err := server.wsConn.ReadMessage()
			if err != nil {
				log.Error(err)
				server.mutex.Unlock()
				server.Close()
				return
			}
			err = server.wsConn.SetReadDeadline(time.Now().Add(server.timeout))
			if err != nil {
				log.Error(err)
				server.mutex.Unlock()
				server.Close()
				return
			}
			go server.handleWsRead(content)
		}
	}
}

func (server *Server) handleWsRead(content []byte) {
	b := make([]byte, len(content))
	copy(b, content)
	server.mutex.Unlock()
	atyp := b[0]
	session := binary.BigEndian.Uint16(b[1:3])
	if server.connPool.isCloseSent(session) {
		return
	}
	switch atyp {
	case 0: //delete
		msg := message.UnpackDelete(b)
		server.delete(msg.Session)
	case 2: //data
		msg := message.UnpackData(b)
		server.receive(msg)
	case socks5.ATYPIPv4, socks5.ATYPDomain, socks5.ATYPIPv6: //request {1,3,4}
		msg := message.UnpackConnect(b)
		wsw := server.getWebsocketWriter(msg.Session)
		var dstAddr net.IP
		var err error
		if msg.Atyp != socks5.ATYPDomain { //{IPv4,IPv6}
			dstAddr = msg.DstAddr
		} else { //Domain
			var dstAddrs []net.IP
			dstAddrs, err = net.LookupIP(string(b[7:]))
			if len(dstAddrs) > 0 {
				dstAddr = dstAddrs[0]
			} else {
				err = errors.New("cannot resolve domain name")
			}
		}
		if err != nil || dstAddr == nil {
			log.WithField("addr", string(b[7:])).Warn(err)
			reply := socks5.NewReply(socks5.RepHostUnreachable, msg.Atyp, net.IPv4zero, []byte{0, 0})
			_, err = wsw.writeReply(*reply)
			if err != nil {
				log.Warn(err)
			}
			return
		}
		var conn net.Conn
		if dstAddr.To4() != nil { //IPv4
			conn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", dstAddr.String(), msg.DstPort))
		} else { //IPv6
			conn, err = net.Dial("tcp", fmt.Sprintf("[%s]:%d", dstAddr.String(), msg.DstPort))
		}
		if err != nil {
			log.WithField("addr", fmt.Sprintf("%s:%d", dstAddr.String(), msg.DstPort)).Error(err)
			reply := socks5.NewReply(socks5.RepHostUnreachable, msg.Atyp, net.IPv4zero, []byte{0, 0})
			_, err = wsw.writeReply(*reply)
			if err != nil {
				log.Warn(err)
			}
			return
		}
		_, addr, port, err := socks5.ParseAddress(conn.LocalAddr().String())
		if err != nil {
			log.WithField("addr", conn.LocalAddr().String()).Error(err)
			return
		}
		reply := socks5.NewReply(socks5.RepSuccess, msg.Atyp, addr, port)
		_, err = wsw.writeReply(*reply)
		if err != nil {
			log.WithFields(log.Fields{
				"atyp": msg.Atyp,
				"addr": addr,
				"port": port,
			}).Warn(err)
			return
		}

		server.connPool.set(msg.Session, &conn)
		go server.peer.forward(msg.Session)
	default:
		log.WithField("atyp", atyp).Warn("Unknown type message")
	}
}

// Handler pass config to ServeHTTP()
type Handler struct {
	Conf *config.ConfigGo
	db   *gorm.DB
}

// ServerHTTP accept incoming HTTP request, establish websocket connections, and a new server for handling the connection. If authentication failed, the request will be redirected to the website set in the configuration file.
func (handler Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	var wsConn *websocket.Conn
	upgrader := websocket.Upgrader{
		EnableCompression: true,
	}
	handler.db = handler.Conf.DB.DB
	err = handler.authenticate(r.Header)
	if err == nil {
		wsConn, err = upgrader.Upgrade(w, r, nil)
	}
	if err != nil {
		log.WithFields(util.Header2Fields(r.Header, r.Body)).Error(err)
		handler.redirect(&w, r)
		return
	}
	wsConn.EnableWriteCompression(true)
	err = wsConn.SetCompressionLevel(flate.BestCompression)
	if err != nil {
		log.Error(err)
		return
	}
	err = wsConn.SetReadDeadline(time.Now().Add(time.Minute))
	//err = wsConn.SetWriteDeadline(time.Now().Add(time.Minute))
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
		log.WithField("token", token).Error(err)
		return err
	}
	if len(data) < 12 {
		err = errors.New("authentication failed: data too short")
		log.WithField("data", data).Error(err)
		return err
	}
	nonce, cipherText := data[:12], data[12:]
	nonceUsed, err := handler.nonceUsed(username, nonce)
	if err != nil {
		log.WithFields(log.Fields{
			"username": username,
			"nonce":    nonce,
		}).Error(err)
		return err
	}
	if nonceUsed {
		err = errors.New("authentication failed in nonce verification")
		log.WithField("nonce", nonce).Error(err)
		return err
	}

	password, err := handler.getPassword(username)
	if err != nil {
		log.WithField("username", username).Error(err)
		return err
	}
	key := argon2.Key([]byte(password), nonce, 3, 32*1024, 4, 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		log.WithField("key", key).Error(err)
		return err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Error(err)
		return err
	}

	plaintext, err := aesgcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"nonce":      nonce,
			"cipherText": cipherText,
		}).Error(err)
		return err
	}

	t := int64(binary.BigEndian.Uint64(plaintext))
	if time.Since(time.Unix(t/1000000000, t%1000000000)).Seconds() > 60 {
		err = errors.New("authentication failed in time verification")
		log.WithField("timestamp", t).Error(err)
		return err
	}
	err = handler.saveNonce(username, nonce)
	if err != nil {
		log.WithFields(log.Fields{
			"username": username,
			"nonce":    nonce,
		}).Error(err)
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
		log.WithField("username", username).Error(err)
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
		log.WithFields(log.Fields{
			"username": username,
			"nonce":    nonce,
		}).Error(err)
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
		log.WithFields(log.Fields{
			"username": username,
			"nonce":    nonce,
		}).Error(err)
		return err
	}
	return nil
}

func (handler Handler) redirect(w *http.ResponseWriter, r *http.Request) {
	newReq, err := http.NewRequest(r.Method, handler.Conf.Server.Pretend.String()+r.RequestURI, r.Body)
	if err != nil {
		log.WithFields(util.Header2Fields(r.Header, r.Body)).Error(err)
		return
	}
	for k, v := range r.Header {
		newReq.Header.Set(k, v[0])
	}
	resp, err := http.DefaultClient.Do(newReq)
	if err != nil {
		log.WithFields(util.Header2Fields(newReq.Header, newReq.Body)).Error(err)
		return
	}
	for k, v := range resp.Header {
		(*w).Header().Set(k, v[0])
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(util.Header2Fields(resp.Header, resp.Body)).Error(err)
		return
	}
	err = resp.Body.Close()
	if err != nil {
		log.Warn(err)
	}
	_, err = (*w).Write(body)
	if err != nil {
		log.WithField("body", body).Warn(err)
		return
	}
}
