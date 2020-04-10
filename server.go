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
	"github.com/iyouport-org/relaybaton/util"
	"github.com/iyouport-org/socks5"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/argon2"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

type Handler struct {
	Conf *config.ConfigGo
	DB   *gorm.DB
}

func (handler Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var wsConn *websocket.Conn
	var err error
	upgrader := websocket.Upgrader{
		EnableCompression: true,
	}
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

	server := Server{
		conn: NewConn(wsConn),
		Conf: handler.Conf,
	}
	server.Run()
}

type Server struct {
	conn *Conn
	Conf *config.ConfigGo
}

func (server *Server) Run() {
	for {
		_, content, err := server.conn.ReadMessage()
		if err != nil {
			log.Error(err)
			err = server.conn.Close()
			if err != nil {
				log.Error(err)
			}
			return
		}
		if len(content) > 3 {
			port := binary.BigEndian.Uint16(content[1:3])
			var connStr string
			switch content[0] {
			case socks5.ATYPIPv4:
				connStr = fmt.Sprintf("%s:%d", net.IP(content[3:]).To4().String(), port)
			case socks5.ATYPIPv6:
				connStr = fmt.Sprintf("[%s]:%d", net.IP(content[3:]).To16().String(), port)
			default:
				//TODO
			}
			outConn, err := net.Dial("tcp", connStr)
			if err != nil {
				log.Error(err)
				_, err = server.conn.Write([]byte{0})
				if err != nil {
					log.Error(err)
					err = server.conn.Close()
					if err != nil {
						log.Error(err)
					}
				}
			} else {
				_, err = server.conn.Write([]byte{1})
				if err != nil {
					log.Error(err)
					err = server.conn.Close()
					if err != nil {
						log.Error(err)
					}
				}
				if server.conn.Run(outConn) != nil {
					return
				}
			}
		}
	}
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
	key := argon2.Key([]byte(password), nonce, 1, 1024, 2, 32)
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
	db := handler.DB
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
	db := handler.DB
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
	db := handler.DB
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
