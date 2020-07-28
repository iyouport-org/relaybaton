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
	"github.com/gorilla/websocket"
	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
	log "github.com/sirupsen/logrus"
	"go.uber.org/fx"
	"net"
	"relaybaton/pkg/config"
	"relaybaton/pkg/socks5"
	"strconv"
)

type Client struct {
	fx.Lifecycle
	*gnet.EventServer
	*config.ConfigGo
	*goroutine.Pool
	conns *Map
}

func NewClient(lc fx.Lifecycle, conf *config.ConfigGo, pool *goroutine.Pool) *Client {
	client := &Client{
		Lifecycle: lc,
		ConfigGo:  conf,
		Pool:      pool,
		conns:     NewMap(),
	}

	client.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Debug("start")
			return nil
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
	key := GetURI(c.RemoteAddr())
	conn, ok := client.conns.Get(key)
	if !ok {
		log.Warn("session do not exist")
		action = gnet.Close
		return
	}
	switch conn.status {
	case StatusOpened:
		mr, err := socks5.NewMethodRequestFrom(frame)
		if err != nil {
			log.Error(err)
			action = gnet.Close
			return
		}
		noAuthRequired := false
		for _, v := range mr.Methods() {
			if v == socks5.MethodNoAuthRequired {
				noAuthRequired = true
				break
			}
		}
		if noAuthRequired {
			mRep := socks5.NewMethodReply(socks5.MethodNoAuthRequired)
			out = mRep.Encode()
		} else {
			log.Error("SOCKS5 error") //TODO
			action = gnet.Close
			return
		}
		conn.status = StatusMethodAccepted
		return out, gnet.None
	case StatusMethodAccepted:
		request, err := socks5.NewRequestFrom(frame)
		if err != nil {
			log.Error(err)
			action = gnet.Close
			return
		}
		conn.dstAddr, err = GetDstAddrFromRequest(request)
		if err != nil {
			log.Error(err)
			action = gnet.Close
			return
		}
		conn.cmd = request.Cmd
		resp, err := conn.DialWs(request)
		if err != nil {
			log.Error(err)
			action = gnet.Close
			return
		}
		repStr := resp.Get("reply")
		repCode, err := strconv.Atoi(repStr)
		if err != nil {
			log.WithField("rep", repStr).Error(err)
			action = gnet.Close
			return
		}
		reply := socks5.NewReply(byte(repCode), socks5.ATypeIPv4, net.IPv4(127, 0, 0, 1).To4(), 1081)
		out = reply.Pack()
		err = client.Submit(conn.Run)
		if err != nil {
			log.Error(err)
			action = gnet.Close
			return
		}
		conn.status = StatusAccepted
		return out, gnet.None
	case StatusAccepted:
		err := conn.remoteConn.WriteMessage(websocket.BinaryMessage, frame)
		if err != nil {
			log.Error(err)
			action = gnet.Close
		}
		return nil, gnet.None
	default:
		//TODO
	}
	return
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
	conn := NewConn(c, client.Clients.Client["2"])
	client.conns.Put(conn)
	return
}

func (client *Client) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	if err != nil {
		log.Error(err)
	}
	key := GetURI(c.RemoteAddr())
	conn, ok := client.conns.Get(key)
	if ok {
		if conn.remoteConn != nil {
			cErr := conn.remoteConn.Close()
			if cErr != nil {
				log.Error(cErr)
			}
		}
	}
	client.conns.Delete(key)
	return
}

func (client *Client) OnShutdown(svr gnet.Server) {
	log.Debug("shutdown")
	client.Pool.Release()
}
