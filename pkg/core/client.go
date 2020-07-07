package core

/*
Client connection status
 - 0 Opened, Method Request Not Read
 - 1 Method Request accepted
 - 2 Request sent
 - 3 Reply received
 - 4 Close sent
*/

import (
	"context"
	"fmt"
	"github.com/Workiva/go-datastructures/queue"
	"github.com/dgrr/fastws"
	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
	log "github.com/sirupsen/logrus"
	"go.uber.org/fx"
	"net"
	"relaybaton/pkg/config"
	"relaybaton/pkg/socks5"
	"sync"
	"time"
)

const (
	StatusOpened         = 0
	StatusMethodAccepted = 1
	StatusRequestSent    = 2
	StatusEstablished    = 3
	StatusCloseSent      = 4
)

type Client struct {
	fx.Lifecycle
	*gnet.EventServer
	*config.ConfigGo
	*goroutine.Pool
	conns        sync.Map //<string,gnet.Conn>
	connStatus   sync.Map //<string,uint8>
	sessionQueue sync.Map //<string,*queue.Queue<[]byte>>
	shutdown     chan gnet.Action
}

func NewClient(lc fx.Lifecycle, conf *config.ConfigGo, pool *goroutine.Pool) *Client {
	client := &Client{
		Lifecycle: lc,
		ConfigGo:  conf,
		Pool:      pool,
		shutdown:  make(chan gnet.Action, 10),
	}
	for k, v := range conf.Clients.Client {
		clientConn := NewClientConn(v)
	CONNECT:
		err := clientConn.Connect()
		if err != nil {
			log.Error(err)
			time.Sleep(5 * time.Second)
			goto CONNECT
		}
		client.conns.Store(k, clientConn)
	}
	client.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			var err error
			/*client.conns.Range(func(key, value interface{}) bool {
				err = client.Submit(func() { //keep alive
					for {
						err = value.(*ClientConn).SendCode(fastws.CodePing, 0, nil)
						if err != nil {
							log.Error(err)
							client.shutdown <- gnet.Shutdown
							return
						}
						time.Sleep(5 * time.Second)
					}
				})
				return true
			})*/

			client.conns.Range(func(key, value interface{}) bool {
				err = client.Submit(func() { //Read websocket
					for {
						_, b, err := value.(*ClientConn).ReadMessage(nil)
						if err != nil {
							log.Error(err)
							client.shutdown <- gnet.Shutdown

							return
						}
						//log.WithField("data", b).Debug("from server")
						packet, err := Unpack(b)
						if err != nil {
							log.WithField("b", b).Error(err)
							//TODO
							return
						}
						status, ok := client.connStatus.Load(string(packet.session))
						if !ok {
							log.Warn("session do not exist")
							continue
						}
						switch packet.PacketType {
						case TypeRequestReply:
							if status == StatusRequestSent { //request sent
								if packet.PacketCmdRep == RepSucceeded {
									c, ok := client.conns.Load(string(packet.session))
									if !ok {
										log.Warn("session do not exist")
										continue
									}
									err := c.(gnet.Conn).AsyncWrite(socks5.NewReply(socks5.RepSucceeded, socks5.ATypeIPv4, net.IPv4zero.To4(), 0).Pack())
									if err != nil {
										log.Error(err)
										continue
									}
								} else {
									log.WithField("Rep", packet.PacketCmdRep).Warn("reply error")
									continue
								}
							} else {
								log.WithField("status", status).Warn("status error")
								continue
							}
							client.connStatus.Store(string(packet.session), StatusEstablished)
						case TypeData:
							if status == StatusEstablished { //Reply received
								c, ok := client.conns.Load(string(packet.session))
								if !ok {
									log.Warn("session do not exist")
									continue
								}
								err := c.(gnet.Conn).AsyncWrite(packet.payload)
								if err != nil {
									log.Error(err)
									continue
								}
							} else {
								log.WithField("status", status).Warn("status error")
								continue
							}
						case TypeClose:
							if status == StatusCloseSent {
								client.connStatus.Delete(string(packet.session))
							} else { //Established
								/*c, ok := client.conns.Load(string(packet.session))
								if !ok {
									log.Warn("session do not exist")
									continue
								}*/

								/*err := c.(gnet.Conn).Close()
								if err != nil {
									log.Error(err)
								}*/

								//TEST

								status, ok := client.connStatus.Load(string(packet.session))
								if !ok {
									log.Error("session not found")
								}
								if status == StatusEstablished {
									p := NewClosePacket(NetWorkTCP, packet.SessATyp, packet.session)
									_, err = client.Select().WriteMessage(fastws.ModeBinary, p.Pack())
									if err != nil {
										log.Error(err)
									}
									client.connStatus.Store(string(packet.session), StatusCloseSent)
								} else {
									log.Error("status error")
								}
								que, ok := client.sessionQueue.Load(string(packet.session))
								if ok {
									que.(*queue.Queue).Dispose()
								}
								client.sessionQueue.Delete(string(packet.session))
								client.conns.Delete(string(packet.session))
								//TEST
							}
						}
					}
				})
				if err != nil {
					log.Error(err)
				}
				return true
			})
			log.Debug("start")
			return err
		},
		OnStop: func(ctx context.Context) error {
			log.Debug("stop")
			return nil
		},
	})
	return client
}

func (client *Client) Run() error {
	err := gnet.Serve(client, fmt.Sprintf("tcp://:%d", client.Clients.Port), gnet.WithMulticore(true), gnet.WithReusePort(true), gnet.WithLogger(log.StandardLogger()))
	if err != nil {
		log.Error(err)
	}
	return err
}

func (client *Client) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	/*log.WithFields(log.Fields{
		"addr": c.RemoteAddr(),
		"raw":  frame,
	}).Info("raw from src")*/
	select {
	case <-client.shutdown:
		return []byte{}, gnet.Shutdown
	default:
		_, session := GetSrcSession(c.RemoteAddr().(*net.TCPAddr))
		status, ok := client.connStatus.Load(string(session))
		if !ok {
			log.Warn("session do not exist")
			return
		}
		if status == StatusCloseSent {
			return []byte{}, gnet.Close
		}
		que, ok := client.sessionQueue.Load(string(session))
		if !ok {
			log.Error("session not found")
			return
		}
		data := make([]byte, len(frame))
		copy(data, frame)
		err := que.(*queue.Queue).Put(data)
		if err != nil {
			log.Error(err)
			return
		}
		//log.WithField("raw", data).Debug("write to chan")
		return
	}
}

func (client *Client) Select() *ClientConn {
	ret, _ := client.conns.Load("2")
	return ret.(*ClientConn)
}

func (client *Client) HandleMethodRequest(data []byte) (b []byte, action gnet.Action) {
	mReq, err := socks5.NewMethodRequestFrom(data)
	if err != nil {
		log.Error(err)
		return nil, gnet.Close
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
	b = mRep.Encode()
	return
}

func (client *Client) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	//log.WithField("addr", c.RemoteAddr()).Debug("connection opened")
	_, session := GetSrcSession(c.RemoteAddr().(*net.TCPAddr))
	client.connStatus.Store(string(session), 0)
	que := queue.New(1024)
	client.sessionQueue.Store(string(session), que)
	client.conns.Store(string(session), c)
	sessATyp, session := GetSrcSession(c.RemoteAddr().(*net.TCPAddr))
	err := client.Submit(func() {
		que, ok := client.sessionQueue.Load(string(session))
		if !ok {
			log.Error("session not found")
			return
		}
		for {
			items, err := que.(*queue.Queue).Get(1024)
			if err != nil {
				//log.Error(err)
				//TODO
				return
			}
			for _, v := range items {
				data := v.([]byte)
				/*log.WithFields(log.Fields{
					"addr": c.RemoteAddr(),
					"data": data,
				}).Info("read from src")*/
				status, ok := client.connStatus.Load(string(session))
				if !ok {
					log.Error("connection not found")
					return
				}
				//log.WithField("status", status).Info("status")
				switch status {
				case StatusOpened: //Method Request Not Read
					var rep []byte
					rep, action = client.HandleMethodRequest(data)
					err := c.AsyncWrite(rep)
					if err != nil {
						log.Error(err)
						return
					}
					client.connStatus.Store(string(session), StatusMethodAccepted)
				case StatusMethodAccepted: //Method Request accepted
					req, err := socks5.NewRequestFrom(data)
					if err != nil {
						log.Error(err)
						return
					}
					conn := client.Select()
					for conn.Conn == nil {
						err = conn.Connect()
						if err != nil {
							log.Error(err)
						}
					}
					//sessATyp, session := GetSrcSession(c.RemoteAddr().(*net.TCPAddr))
					packet := NewRequestPacket(NetWorkTCP, sessATyp, req.Cmd, ATypConvert(req.ATyp), session, req.DstAddr, req.DstPort)
					//log.Debug(packet.Pack())
					_, err = conn.WriteMessage(fastws.ModeBinary, packet.Pack())
					if err != nil {
						log.Error(err)
					}
					client.connStatus.Store(string(session), StatusRequestSent)
				case StatusRequestSent: //request sent
				//TODO
				case StatusEstablished: //reply received
					packet := NewDataPacket(NetWorkTCP, sessATyp, session, data)
					conn := client.Select()
					for conn.Conn == nil {
						err := conn.Connect()
						if err != nil {
							log.Error(err)
						}
					}
					_, err := conn.WriteMessage(fastws.ModeBinary, packet.Pack())
					if err != nil {
						log.Error(err)
					}
					//log.WithField("data", packet.payload).Info("write to server")
				default:

				}
			}
		}
	})
	if err != nil {
		log.Error(err)
	}
	return
}

func (client *Client) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	if err != nil {
		log.Error(err)
	}
	//log.WithField("addr", c.RemoteAddr()).Debug("connection closed")
	sessATyp, session := GetSrcSession(c.RemoteAddr().(*net.TCPAddr))
	status, ok := client.connStatus.Load(string(session))
	if !ok {
		log.Error("session not found")
	}
	if status == StatusEstablished {
		p := NewClosePacket(NetWorkTCP, sessATyp, session)
		_, err = client.Select().WriteMessage(fastws.ModeBinary, p.Pack())
		if err != nil {
			log.Error(err)
		}
		client.connStatus.Store(string(session), StatusCloseSent)
	} else {
		log.WithField("status", status).Error("status error")
	}
	que, ok := client.sessionQueue.Load(string(session))
	if ok {
		que.(*queue.Queue).Dispose()
	}
	client.sessionQueue.Delete(string(session))
	client.conns.Delete(string(session))
	return
}

func (client *Client) OnShutdown(svr gnet.Server) {
	log.Debug("shutdown")
	client.Pool.Release()
}
