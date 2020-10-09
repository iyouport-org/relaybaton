package core

import (
	"compress/flate"
	"context"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/emirpasic/gods/maps/hashmap"
	"github.com/fasthttp/websocket"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/iyouport-org/relaybaton/internal/memsocket"
	"github.com/iyouport-org/relaybaton/pkg/config"
	"github.com/iyouport-org/relaybaton/pkg/model"
	"github.com/iyouport-org/relaybaton/pkg/socks5"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/reuseport"
	"go.uber.org/fx"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm/clause"
)

type Server struct {
	fx.Lifecycle
	net.Listener
	*config.ConfigGo
	*hashmap.Map
	mutex sync.RWMutex
	ms    *memsocket.MemSocket
}

func NewServer(lc fx.Lifecycle, conf *config.ConfigGo) *Server {
	server := &Server{
		Lifecycle: lc,
		ConfigGo:  conf,
		Map:       hashmap.New(),
		ms:        memsocket.NewMemSocket(),
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
	gin.SetMode(gin.DebugMode)
	gin.DefaultErrorWriter = log.StandardLogger().WriterLevel(log.ErrorLevel)
	gin.DefaultWriter = log.StandardLogger().Writer()
	sessionTime := make([]byte, 8)
	sessionRandom := make([]byte, 16)
	authKey := make([]byte, 64)
	cryptKey := make([]byte, 32)
	n, err := rand.Read(authKey)
	if err != nil {
		log.Error(err)
		return
	}
	if n != 64 {
		err = errors.New("auth key not generated")
		log.WithField("n", n).Error(err)
		return
	}
	n, err = rand.Read(cryptKey)
	if err != nil {
		log.Error(err)
		return
	}
	if n != 32 {
		err = errors.New("crypt key not generated")
		log.WithField("n", n).Error(err)
		return
	}
	n, err = rand.Read(sessionRandom)
	if err != nil {
		log.Error(err)
		return
	}
	if n != 16 {
		err = errors.New("sessionName key not generated")
		log.WithField("n", n).Error(err)
		return
	}
	binary.BigEndian.PutUint64(sessionTime, uint64(time.Now().UnixNano()))

	r := gin.Default()
	r.LoadHTMLFiles("web/index.html")
	r.Use(static.Serve("/", static.LocalFile("./web", false)))
	r.Use(sessions.Sessions(hex.EncodeToString(sha512.New512_256().Sum(append(sessionTime, sessionRandom...))), cookie.NewStore(authKey, cryptKey)))
	r.Use(gzip.Gzip(gzip.BestCompression))
	r.GET("/", server.ServeRoot)
	r.GET("/captcha/:hash", server.GetCaptcha)

	r.POST("/user", server.PostUser)
	r.DELETE("/user/:id", server.DeleteUser)
	r.PUT("/user/:id", server.PutUser)
	r.GET("/user", server.GetUser)
	r.GET("/user/:id", server.GetUserOne)

	r.POST("/session", server.PostSession)
	r.DELETE("/session", server.DeleteSession)
	r.PUT("/session", server.PutSession)
	r.GET("/session", server.GetSession)

	r.POST("/log", server.PostLog)
	r.DELETE("/log/:id", server.DeleteLog)
	r.PUT("/log/:id", server.PutLog)
	r.GET("/log/:id", server.GetLogOne)
	r.GET("/log", server.GetLogList)

	r.POST("/config", server.PostConfig)
	r.DELETE("/config", server.DeleteConfig)
	r.PUT("/config", server.PutConfig)
	r.GET("/config", server.GetConfig)

	r.POST("/plan", server.PostPlan)
	r.DELETE("/plan/:id", server.DeletePlan)
	r.PUT("/plan/:id", server.PutPlan)
	r.GET("/plan/:id", server.GetPlanOne)
	r.GET("/plan", server.GetPlan)

	r.POST("/meta", server.PostMeta)
	r.DELETE("/meta", server.DeleteMeta)
	r.PUT("/meta/undefined", server.PutMeta)
	r.GET("/meta/undefined", server.GetMeta)

	r.POST("/notice", server.PostNotice)
	r.DELETE("/notice/:id", server.DeleteNotice)
	r.PUT("/notice/:id", server.PutNotice)
	r.GET("/notice/:id", server.GetNoticeOne)
	r.GET("/notice", server.GetNotice)

	go func() {
		err := r.RunListener(server.ms.Listener())
		if err != nil {
			log.Error(err)
		}
	}()
	ln, err := reuseport.Listen("tcp4", ":80")
	if err != nil {
		log.Fatal("error in reuseport listener: %s", err)
	}
	defer ln.Close()
	if err = fasthttp.Serve(ln, server.requestHandler); err != nil {
		log.Fatal("error in fasthttp Server: %s", err)
	}
	if err != nil {
		log.Error(err)
	}
}

func (server *Server) requestHandler(ctx *fasthttp.RequestCtx) {
	if !server.Authenticate(ctx) {
		server.serveWeb(ctx)
		return
	}
	var upgrader = websocket.FastHTTPUpgrader{
		EnableCompression: true,
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", string(ctx.Request.Header.Peek("addr")))
	if err != nil {
		log.Error(err)
		return
	}
	ip := tcpAddr.IP
	if ip.To4() == nil {
		ip = ip.To4()
	} else {
		ip = ip.To16()
	}
	if isReservedIP(ip) {
		err := errors.New("reserved")
		log.WithField("ip", ip.String()).Error(err)
		return
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
		server.serveWeb(ctx)
		log.Println(err)
		return
	}
}

func (server *Server) Authenticate(ctx *fasthttp.RequestCtx) bool {
	if ctx.Request.Header.Peek("addr") == nil || ctx.Request.Header.Peek("username") == nil || ctx.Request.Header.Peek("password") == nil {
		return false
	}
	username := string(ctx.Request.Header.Peek("username"))
	password := string(ctx.Request.Header.Peek("password"))
	user, err := server.getUser(username)
	if err != nil {
		log.Error(err)
		return false
	}
	if username == "admin" {
		return password == user.Password
	} else {
		if user.Plan.TrafficLimit <= user.TrafficUsed {
			log.WithFields(log.Fields{
				"plan":  user.Plan.Name,
				"limit": user.Plan.TrafficLimit,
				"used":  user.TrafficUsed,
			}).Debug("traffic running out")
			return false
		}
		sha512key, err := base64.StdEncoding.DecodeString(password)
		if err != nil {
			log.Error(err)
			return false
		}
		if len(sha512key) != 64 {
			log.Debug(sha512key)
			return false
		}
		correctKey, err := base64.StdEncoding.DecodeString(user.Password)
		if err != nil {
			log.Error(err)
			return false
		}
		err = bcrypt.CompareHashAndPassword(correctKey, sha512key)
		if err != nil {
			log.Error(err)
			return false
		}
		return true
	}
}

func (server *Server) getUser(username string) (*model.User, error) {
	db := server.DB.DB
	user := &model.User{}
	err := db.Preload(clause.Associations).Where("username = ?", username).First(user).Error
	if err != nil {
		log.WithField("username", username).Error(err)
		return nil, err
	}
	return user, err
}

/**
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
**/

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
		log.WithField("limit", user.Plan.BandwidthLimit).Debug("bandwidth limit")
		bucket := NewRateLimiter(user.Plan.BandwidthLimit / 2)
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
