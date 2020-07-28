package core

import (
	"compress/flate"
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"go.uber.org/fx"
	"net"
	"net/http"
	"relaybaton/pkg/config"
	"relaybaton/pkg/socks5"
)

type Server struct {
	fx.Lifecycle
	net.Listener
	*config.ConfigGo
}

func NewServer(lc fx.Lifecycle, conf *config.ConfigGo) *Server {
	server := &Server{
		Lifecycle: lc,
		ConfigGo:  conf,
	}
	server.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Debug("server start")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Debug("server shutdown")
			return nil
		},
	})
	return server
}

func (server *Server) Run() {
	err := http.ListenAndServe(":80", server)
	if err != nil {
		log.Error(err)
	}
}

func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !server.Authenticate(r) {
		return
	}
	var upgrader = websocket.Upgrader{
		EnableCompression: true,
	}
	c, err := net.Dial("tcp", r.Header.Get("addr"))
	if err != nil {
		log.Error(err)
		return
	}
	header := http.Header{}
	header.Add("reply", fmt.Sprintf("%d", socks5.RepSucceeded))
	conn, err := upgrader.Upgrade(w, r, header)
	if err != nil {
		log.Error(err)
		return
	}
	err = conn.SetCompressionLevel(flate.BestCompression)
	if err != nil {
		log.Error(err)
		conn.Close()
		return
	}

	go func() {
		for {
			b := make([]byte, 65535)
			n, err := c.Read(b)
			//_, err = io.Copy(writer, c)
			if err != nil {
				log.Error(err)
				conn.Close()
				return
			}
			err = conn.WriteMessage(websocket.BinaryMessage, b[:n])
			if err != nil {
				log.Error(err)
				conn.Close()
				return
			}
		}
	}()

	for {
		_, b, err := conn.ReadMessage()
		if err != nil {
			log.Error(err)
			conn.Close()
			return
		}
		_, err = c.Write(b)
		if err != nil {
			log.Error(err)
			conn.Close()
			return
		}
	}
}

func (server *Server) Authenticate(r *http.Request) bool {
	if r.Header.Get("addr") == "" {
		log.Debug("false")
		return false
	}
	return true
}
