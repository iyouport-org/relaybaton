package core

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/Workiva/go-datastructures/queue"
	"github.com/dgrr/fastws"
	"github.com/fasthttp/websocket"
	"github.com/panjf2000/ants/v2"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/reuseport"
	"go.uber.org/fx"
	"net"
	"relaybaton/pkg/config"
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
	var err error
	server.Listener, err = reuseport.Listen("tcp4", fmt.Sprintf(":%d", server.Server.Port))
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()
	err = fasthttp.Serve(server, server.HandleFastHTTP)
	if err != nil {
		log.Error(err)
	}
	/*
		err = fasthttp.Serve(server, fastws.Upgrade(server.FastWsHandler))
		if err != nil {
			log.Error(err)
		}*/
}

func (server *Server) HandleFastHTTP(ctx *fasthttp.RequestCtx) {
	if !websocket.FastHTTPIsWebSocketUpgrade(ctx) {
		log.Error("not websocket")
		ctx.NotFound()
		return
	}

	(&fastws.Upgrader{
		UpgradeHandler: func(ctx *fasthttp.RequestCtx) bool {
			return true
		},
		Handler: server.FastWsHandler,
	}).Upgrade(ctx)
}

func (server *Server) FastWsHandler(conn *fastws.Conn) {
	conn.Mode = fastws.ModeBinary
	var conns sync.Map
	var connStatus sync.Map
	var sessionChan sync.Map

	poolForward, err := ants.NewPoolWithFunc(65535, func(payload interface{}) {
		for {
			p := payload.(struct {
				tcpConn  *net.TCPConn
				sessATyp PacketATyp
				session  []byte
			})
			size := 32 * 1024
			b := make([]byte, size)
			n, err := p.tcpConn.Read(b)
			if err != nil { //Close
				log.Error(err)

				que, ok := sessionChan.Load(string(p.session))
				if !ok {
					log.Warn("session not found")
					return
				} else {
					que.(*queue.Queue).Dispose()
				}
				sessionChan.Delete(string(p.session))

				err = p.tcpConn.Close()
				if err != nil {
					log.Error(err)
				}
				conns.Delete(string(p.session))

				packet := NewClosePacket(NetWorkTCP, p.sessATyp, p.session)
				_, err := conn.WriteMessage(fastws.ModeBinary, packet.Pack())
				if err != nil {
					log.Error(err)
				}
				_, ok = connStatus.Load(string(p.session))
				if !ok {
					log.Warn("session not found")
					return
				}
				connStatus.Store(string(p.session), StatusCloseSent)
				return
			}
			packet := NewDataPacket(NetWorkTCP, p.sessATyp, p.session, b[:n])
			_, err = conn.WriteMessage(fastws.ModeBinary, packet.Pack())
			if err != nil {
				log.Error(err)
				return
			}
			//log.WithField("payload", packet.payload).Debug("write to client")
		}
	})
	if err != nil {
		log.Error(err)
		return
	}
	defer poolForward.Release()

	poolHandle, err := ants.NewPool(1024)
	if err != nil {
		log.Error(err)
		return
	}
	defer poolHandle.Release()

	poolReceive, err := ants.NewPoolWithFunc(65535, func(i interface{}) { //forward
		que, ok := sessionChan.Load(i.(string))
		if !ok {
			//TODO
			log.Error("session not found")
			return
		}
		for {
			items, err := que.(*queue.Queue).Get(1024)
			if err != nil {
				log.Error(err)
				//TODO
				return
			}
			for _, v := range items {
				data := v.([]byte)
				tcpConn, ok := conns.Load(i.(string))
				if !ok {
					log.Error("session not found")
					return
				}
				_, err := tcpConn.(*net.TCPConn).Write(data)
				if err != nil { //Close
					log.Error(err)
					que.(*queue.Queue).Dispose()
					sessionChan.Delete(i.(string))
					err := tcpConn.(*net.TCPConn).Close()
					if err != nil {
						log.Error(err)
						break
					}
					conns.Delete(i.(string))
					session := []byte(i.(string))
					var sessATyp PacketATyp
					if len(session) == 6 {
						sessATyp = ATypIPv4
					} else {
						sessATyp = ATypIPv6
					}
					packet := NewClosePacket(NetWorkTCP, sessATyp, []byte(i.(string)))
					_, err = conn.WriteMessage(fastws.ModeBinary, packet.Pack())
					if err != nil {
						log.Error(err)
					}
					_, ok := connStatus.Load(i.(string))
					if !ok {
						log.Warn("session not found")
						return
					}
					connStatus.Store(i.(string), StatusCloseSent)
					return
				}
				//log.WithField("data", data).Debug("data write to dst")
			}
		}
	})
	defer poolReceive.Release()

	/*err = poolHandle.Submit(func() {
		for {
			err = conn.SendCode(fastws.CodePong, 0, nil)
			if err != nil {
				log.Error(err)
				return
			}
			time.Sleep(5 * time.Second)
		}
	})
	if err != nil {
		log.Error(err)
	}*/

	for {
		_, b, err := conn.ReadMessage(nil)
		if err != nil {
			log.Error(err)
			//TODO
			return
		}
		data := make([]byte, len(b))
		copy(data, b)
		packet, err := Unpack(data)
		if err != nil {
			log.WithField("b", b).Error(err)
			//TODO
			return
		}
		//log.WithField("data", packet.payload).Debug("read from client")
		if len(data) < 9 {
			continue
		}

		que, loaded := sessionChan.LoadOrStore(string(packet.session), queue.New(1024))
		if loaded { //Not New
			if packet.PacketType == TypeData {
				err = que.(*queue.Queue).Put(packet.payload)
				if err != nil {
					log.Error(err)
					continue
				}
			} else if packet.PacketType == TypeClose {
				status, ok := connStatus.Load(string(packet.session))
				if !ok {
					log.Error("session not found")
					continue
				}
				connStatus.Delete(string(packet.session))
				if status == StatusEstablished {
					tcpConn, ok := conns.Load(string(packet.session))
					if !ok {
						log.Error("session not found")
						continue
					}
					conns.Delete(string(packet.session))
					queClose, ok := sessionChan.Load(string(packet.session))
					if !ok {
						log.Error("session not found")
						continue
					}
					sessionChan.Delete(string(packet.session))
					err := tcpConn.(*net.TCPConn).Close()
					if err != nil {
						log.Error(err)
						continue
					}
					queClose.(*queue.Queue).Dispose()
					packetClose := NewClosePacket(NetWorkTCP, packet.SessATyp, packet.session)
					_, err = conn.WriteMessage(fastws.ModeBinary, packetClose.Pack())
					if err != nil {
						log.Error(err)
						continue
					}
				} else {
					log.Error("status error")
				}
			} else {
				//TODO
				log.Error("type error")
				continue
			}
		} else { //New
			err = poolHandle.Submit(func() {
				if packet.PacketType == TypeRequestReply {
					tcpConn, err := net.DialTCP("tcp", nil, &net.TCPAddr{
						IP:   packet.payload[2:],
						Port: int(binary.BigEndian.Uint16(packet.payload[0:2])),
					})
					if err != nil {
						log.Error(err)
						return
					}
					reply := NewReplyPacket(packet.PacketNetwork, packet.SessATyp, RepSucceeded, ATypConvert(ATypIPv4), packet.session, net.IPv4zero.To4(), 0)
					_, err = conn.WriteMessage(fastws.ModeBinary, reply.Pack())
					if err != nil {
						log.Error(err)
						return
					}
					//log.WithField("data", reply.Pack()).Debug("write to client")
					err = poolForward.Invoke(struct {
						tcpConn  *net.TCPConn
						sessATyp PacketATyp
						session  []byte
					}{tcpConn: tcpConn,
						sessATyp: packet.SessATyp,
						session:  packet.session})
					if err != nil {
						log.Error(err)
						return
					}
					conns.Store(string(packet.session), tcpConn)
					connStatus.Store(string(packet.session), StatusEstablished)
					err = poolReceive.Invoke(string(packet.session))
					if err != nil {
						log.Error(err)
						return
					}
				} else {
					//TODO
					log.Error("wrong type")
				}
			})
			if err != nil {
				log.Error(err)
			}
		}
	}
}
