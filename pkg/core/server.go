package core

import (
	"compress/flate"
	"context"
	"errors"
	"fmt"
	"github.com/emirpasic/gods/maps/hashmap"
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
	*hashmap.Map
	mutex sync.RWMutex
}

func NewServer(lc fx.Lifecycle, conf *config.ConfigGo) *Server {
	server := &Server{
		Lifecycle: lc,
		ConfigGo:  conf,
		Map:       hashmap.New(),
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
	username := ctx.Request.Header.Peek("username")
	err = upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		err = conn.SetCompressionLevel(flate.BestCompression)
		if err != nil {
			log.Error(err)
			conn.Close()
			return
		}
		var wg sync.WaitGroup
		wg.Add(2)
		bandwidth := 0
		defer func(username string) {
			user, err := server.getUser(username)
			if err != nil {
				log.Error(err)
				return
			}
			user.TrafficUsed += uint(bandwidth)
			db := server.DB.DB
			db.AutoMigrate(&User{})
			db.Save(user)
		}(string(username))
		go func(username string) {
			for {
				bucket, err := server.GetBucket(username)
				if err != nil {
					log.Error(err)
					wg.Done()
					return
				}
				readLen := bucket.Available()
				if readLen == 0 {
					readLen = uint64(bucket.bandwidth)
				}
				b := make([]byte, readLen)
				n, err := c.Read(b)
				if err != nil {
					log.Error(err)
					wg.Done()
					return
				}
				err = bucket.Wait(uint(n))
				if err != nil {
					log.Error(err)
					wg.Done()
					return
				}
				bandwidth += n
				err = conn.WriteMessage(websocket.BinaryMessage, b[:n])
				if err != nil {
					log.Error(err)
					wg.Done()
					return
				}
			}
		}(string(username))
		go func(username string) {
			for {
				_, b, err := conn.ReadMessage()
				if err != nil {
					log.Error(err)
					wg.Done()
					return
				}
				bandwidth += len(b)
				_, err = c.Write(b)
				if err != nil {
					log.Error(err)
					wg.Done()
					return
				}
			}
		}(string(username))
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
	user, err := server.getUser(username)
	if err != nil {
		log.Error(err)
		return false
	}
	return (password == user.Password) && (user.TrafficLimit > user.TrafficUsed)
}

func (server *Server) getUser(username string) (*User, error) {
	db := server.DB.DB
	user := &User{}
	db.AutoMigrate(user)
	err := db.Where("username = ?", username).First(user).Error
	if err != nil {
		log.WithField("username", username).Error(err)
		return nil, err
	}
	return user, err
}

func (server *Server) redirect(ctx *fasthttp.RequestCtx) {
	newReq := fasthttp.AcquireRequest()
	ctx.Request.CopyTo(newReq)
	newReq.SetHost(server.Server.Pretend.Host)
	newReq.SetRequestURI(server.Server.Pretend.String() + string(ctx.Request.RequestURI()))
	newReq.WriteTo(log.StandardLogger().Writer())
	rep := fasthttp.AcquireResponse()
	err := fasthttp.Do(newReq, rep)
	newReq.Header.Header()
	rep.CopyTo(&ctx.Response)
	if err != nil {
		log.Error(err)
		return
	}
}

func (server *Server) GetBucket(username string) (*RateLimiter, error) {
	server.mutex.RLock()
	v, ok := server.Map.Get(username)
	server.mutex.RUnlock()
	if !ok {
		user, err := server.getUser(username)
		if err != nil {
			log.WithField("username", username).Error(err)
			return nil, err
		}
		log.WithField("limit", user.BandwidthLimit).Debug("bandwidth limit")
		bucket := NewRateLimiter(user.BandwidthLimit / 2)
		server.mutex.Lock()
		server.Map.Put(username, bucket)
		server.mutex.Unlock()
		log.Debug("bucket saved")
		return bucket, nil
	}
	bucket, ok := v.(*RateLimiter)
	if !ok {
		err := errors.New("type error")
		log.Error(err)
		return nil, err
	}
	return bucket, nil
}
