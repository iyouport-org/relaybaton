package core

import (
	"context"
	"fmt"
	"github.com/dgrr/fastws"
	"github.com/fasthttp/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/reuseport"
	"go.uber.org/fx"
	"net"
	"relaybaton-dev/pkg/config"
)

type Server struct {
	lc   fx.Lifecycle
	ln   net.Listener
	conf *config.ConfigGo
}

func NewServer(lc fx.Lifecycle, conf *config.ConfigGo) *Server {
	server := &Server{
		conf: conf,
	}
	lc.Append(fx.Hook{
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
	var err error
	server.ln, err = reuseport.Listen("tcp4", fmt.Sprintf(":%d", server.conf.Server.Port))
	if err != nil {
		log.Fatal(err)
	}
	defer server.ln.Close()
	err = fasthttp.Serve(server.ln, server.HandleFastHTTP)
	if err != nil {
		log.Error(err)
	}

	err = fasthttp.Serve(server.ln, fastws.Upgrade(server.FastWsHandler))
	if err != nil {
		log.Error(err)
	}
}

func (server *Server) HandleFastHTTP(ctx *fasthttp.RequestCtx) {
	if !websocket.FastHTTPIsWebSocketUpgrade(ctx) {
		log.Error("not websocket")
		return
	}

	(&fastws.Upgrader{
		UpgradeHandler: func(ctx *fasthttp.RequestCtx) bool {
			return true
		},
		Handler: server.FastWsHandler,
	}).Upgrade(ctx)

	/*
		var upgrader = websocket.FastHTTPUpgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}

		err := upgrader.Upgrade(ctx, server.wsHandler)
		if err != nil {
			log.Error(err)
			return
		}*/
}

func (server *Server) wsHandler(conn *websocket.Conn) {
	t, p, err := conn.ReadMessage()
	if err != nil {
		log.Error(err)
		return
	}
	log.Debug(p)
	err = conn.WriteMessage(t, p)
	if err != nil {
		log.Error(err)
		return
	}
}

func (server *Server) FastWsHandler(conn *fastws.Conn) {
	//b:=make([]byte,1024)
	conn.Mode = fastws.ModeBinary
	_, b, err := conn.ReadMessage(nil)
	if err != nil {
		log.Error(err)
		return
	}
	log.Debug(b)
	_, err = conn.Write(b)
	if err != nil {
		log.Error(err)
		return
	}
}
