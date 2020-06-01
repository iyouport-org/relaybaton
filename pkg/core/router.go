package core

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"github.com/dgrr/fastws"
	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
	log "github.com/sirupsen/logrus"
	"go.uber.org/fx"
	"net"
	"relaybaton-dev/pkg/config"
	"relaybaton-dev/pkg/socks5"
	"sync"
)

type Router struct {
	*gnet.EventServer
	lc         *fx.Lifecycle
	pool       *goroutine.Pool
	conf       *config.ConfigGo
	coonStatus sync.Map
	coolPool   sync.Map
}

func NewRouter(lc fx.Lifecycle, conf *config.ConfigGo, pool *goroutine.Pool) *Router {
	router := &Router{
		lc:   &lc,
		conf: conf,
		pool: pool,
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			//go router.Run()
			log.Debug("start")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Debug("stop")
			return nil
		},
	})
	return router
}

func (router *Router) Run() error {
	err := gnet.Serve(router, fmt.Sprintf("tcp://:%d", router.conf.Clients.Port), gnet.WithMulticore(true), gnet.WithReusePort(true), gnet.WithLogger(log.StandardLogger()))
	if err != nil {
		log.Error(err)
	}
	return err
}

func (router *Router) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	data := append([]byte{}, frame...)
	err := router.pool.Submit(func() {
		log.Info(data)
		log.Info(c.RemoteAddr())
		status, ok := router.coonStatus.Load(c.RemoteAddr())
		if !ok {
			log.Error("connection not found")
			action = gnet.Close
			return
		}
		switch status { //Method Request Not Read
		case 0:
			mReq, err := socks5.NewMethodRequestFrom(data)
			if err != nil {
				log.Error(err)
				action = gnet.Close
				return
			}
			noAuth := false
			for _, v := range mReq.Methods() {
				if v == socks5.MethodNoAuthRequired {
					noAuth = true
				}
			}
			var mRep socks5.MethodReply
			if noAuth {
				mRep = socks5.NewMethodReply(socks5.MethodNoAuthRequired)
			} else {
				mRep = socks5.NewMethodReply(socks5.MethodNoAuthRequired)
			}
			err = c.AsyncWrite(mRep.Encode())
			if err != nil {
				log.Error(err)
				action = gnet.Close
				return
			}
			router.coonStatus.Store(c.RemoteAddr(), 1)
		case 1:
			req, err := socks5.NewRequestFrom(data)
			if err != nil {
				log.Error(err)
				action = gnet.Close
				return
			}
			log.WithFields(log.Fields{
				"cmd":     req.Cmd(),
				"aTyp":    req.ATyp(),
				"dstAddr": req.DstAddr(),
				"dstPort": req.DstPort(),
			}).Debug()
			servername := "example.com" //TODO Select client
			esnikey, err := getESNIKey(servername)
			if err != nil {
				log.Error(err)
				action = gnet.Close
				return
			}
			tlsConn, err := tls.Dial("tcp", servername+":443", &tls.Config{
				ServerName:     servername + ":443",
				ClientESNIKeys: esnikey,
			})
			if err != nil {
				log.Error(err)
				action = gnet.Close
				return
			}
			conn, err := fastws.Client(tlsConn, "wss://"+servername)
			if err != nil {
				log.Error(err)
				action = gnet.Close
				return
			}
			log.Debug("websocket dialed")
			conn.Mode = fastws.ModeBinary
			_, err = conn.Write([]byte{'h', 'e', 'l', 'l', 'o'})
			//err = conn.WriteMessage(websocket.BinaryMessage, []byte{'h', 'e', 'l', 'l', 'o'})
			if err != nil {
				log.Error(err)
			}
			_, b, err := conn.ReadMessage(nil)
			if err != nil {
				log.Error(err)
			}
			log.Debug(b)
		case 2:
		default:

		}
	})
	if err != nil {
		log.Error(err)
	}
	return
}

func (router *Router) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	log.WithField("addr", c.RemoteAddr()).Debug("connection opened")
	router.coonStatus.Store(c.RemoteAddr(), 0)
	return
}

func (router *Router) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	if err != nil {
		log.Error(err)
	}
	log.WithField("addr", c.RemoteAddr()).Debug("connection closed")
	router.coonStatus.Delete(c.RemoteAddr())
	router.coolPool.Delete(c.RemoteAddr())
	return
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
