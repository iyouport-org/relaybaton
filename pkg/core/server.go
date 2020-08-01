package core

import (
	"compress/flate"
	"context"
	"fmt"
	"github.com/fasthttp/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/reuseport"
	"go.uber.org/fx"
	"net"
	"relaybaton/pkg/config"
	"relaybaton/pkg/socks5"
	"sync"
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
	server.DB.DB.AutoMigrate(&User{})
	ln, err := reuseport.Listen("tcp4", ":80")
	if err != nil {
		log.Fatal("error in reuseport listener: %s", err)
	}
	if err = fasthttp.Serve(ln, server.requestHandler); err != nil {
		log.Fatal("error in fasthttp Server: %s", err)
	}
	if err != nil {
		log.Error(err)
	}
}

func (server *Server) requestHandler(ctx *fasthttp.RequestCtx) {
	if !server.Authenticate(ctx) {
		server.redirect(ctx)
		return
	}
	var upgrader = websocket.FastHTTPUpgrader{
		EnableCompression: true,
	}
	c, err := net.Dial("tcp", string(ctx.Request.Header.Peek("addr")))
	if err != nil {
		log.Error(err)
		return
	}
	ctx.Response.Header.Add("reply", fmt.Sprintf("%d", socks5.RepSucceeded))
	err = upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		err = conn.SetCompressionLevel(flate.BestCompression)
		if err != nil {
			log.Error(err)
			conn.Close()
			return
		}
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			for {
				b := make([]byte, 65535)
				n, err := c.Read(b)
				//_, err = io.Copy(writer, c)
				if err != nil {
					log.Error(err)
					wg.Done()
					return
				}
				err = conn.WriteMessage(websocket.BinaryMessage, b[:n])
				if err != nil {
					log.Error(err)
					wg.Done()
					return
				}
			}
		}()

		go func() {
			for {
				_, b, err := conn.ReadMessage()
				if err != nil {
					log.Error(err)
					wg.Done()
					return
				}
				_, err = c.Write(b)
				if err != nil {
					log.Error(err)
					wg.Done()
					return
				}
			}
		}()
		wg.Wait()
		err := conn.Close()
		if err != nil {
			log.Error(err)
		}
	})
	if err != nil {
		log.Println(err)
		return
	}
}

func (server *Server) Authenticate(ctx *fasthttp.RequestCtx) bool {
	if ctx.Request.Header.Peek("addr") == nil || ctx.Request.Header.Peek("username") == nil || ctx.Request.Header.Peek("password") == nil {
		log.Debug("false")
		return false
	}
	username := string(ctx.Request.Header.Peek("username"))
	password := string(ctx.Request.Header.Peek("password"))
	correctPassword, err := server.getPassword(username)
	if err != nil {
		log.Error(err)
		return false
	}
	return password == correctPassword
}

func (server *Server) getPassword(username string) (string, error) {
	db := server.DB.DB
	db.AutoMigrate(&User{})
	var user User
	err := db.Where("username = ?", username).First(&user).Error
	if err != nil {
		log.WithField("username", username).Error(err)
		return "", err
	}
	return user.Password, nil
}

func (server *Server) redirect(ctx *fasthttp.RequestCtx) {
	newReq := fasthttp.AcquireRequest()
	ctx.Request.CopyTo(newReq)
	newReq.Header.SetHost(server.Server.Pretend.String() + string(ctx.Request.RequestURI()))
	err := fasthttp.Do(newReq, &ctx.Response)
	if err != nil {
		log.Error(err)
		return
	}
}
