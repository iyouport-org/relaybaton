package core

import (
	"fmt"
	"io"
	"net"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"github.com/valyala/fasthttp/reuseport"
	"golang.org/x/net/proxy"
)

type HTTPServer struct {
	Client *Client
	once   sync.Once
}

type hijackHandler struct {
	server  *HTTPServer
	connStr string
}

func (server *HTTPServer) Serve() error {
	ln4, err := reuseport.Listen("tcp4", fmt.Sprintf(":%d", server.Client.Client.HTTPPort))
	if err != nil {
		log.Error(err)
		return err
	}
	ln6, err := reuseport.Listen("tcp6", fmt.Sprintf(":%d", server.Client.Client.HTTPPort))
	if err != nil {
		log.Error(err)
		return err
	}
	go func() {
		err = fasthttp.Serve(ln4, server.requestHandler)
		if err != nil {
			log.Error(err)
		}
	}()
	go func() {
		err = fasthttp.Serve(ln6, server.requestHandler)
		if err != nil {
			log.Error(err)
		}
	}()

	return nil
}

func (server *HTTPServer) requestHandler(ctx *fasthttp.RequestCtx) {
	if ctx.IsConnect() {
		handler := hijackHandler{
			server:  server,
			connStr: string(ctx.Request.RequestURI()),
		}
		ctx.Hijack(handler.Handle)
	} else {
		c := &fasthttp.Client{
			Dial: fasthttpproxy.FasthttpSocksDialer(fmt.Sprintf("localhost:%d", server.Client.Client.Port)),
		}
		err := c.Do(&ctx.Request, &ctx.Response)
		if err != nil {
			log.Error(err)
		}
	}
}

func (handler *hijackHandler) Handle(c net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)
	dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("localhost:%d", handler.server.Client.Client.Port), nil, nil)
	if err != nil {
		log.Error(err)
		return
	}
	s5conn, err := dialer.Dial("tcp", handler.connStr)
	if err != nil {
		log.Error(err)
		return
	}
	go func() {
		defer wg.Done()
		_, err := io.Copy(s5conn, c)
		if err != nil {
			log.Error(err)
		}
	}()
	go func() {
		defer wg.Done()
		_, err := io.Copy(c, s5conn)
		if err != nil {
			log.Error(err)
		}
	}()
	wg.Wait()
}
