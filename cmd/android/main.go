package android

import (
	"context"
	"github.com/eycorsican/go-tun2socks/core"
	"github.com/eycorsican/go-tun2socks/proxy/socks"
	"github.com/panjf2000/gnet/pool/goroutine"
	log "github.com/sirupsen/logrus"
	"go.uber.org/fx"
	_ "golang.org/x/mobile/bind"
	_ "golang.org/x/mobile/bind/java"
	"relaybaton/pkg/config"
	relaybatonCore "relaybaton/pkg/core"
	"time"
)

var lwipStack core.LWIPStack

type Android struct {
	*relaybatonCore.Client
	*fx.App
}

type PacketFlow interface {
	WritePacket(packet []byte)
}

func (android *Android) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	android.App = fx.New(
		fx.Provide(
			relaybatonCore.NewClient,
			config.NewConfClient,
			goroutine.Default,
		),
		fx.Logger(log.StandardLogger()),
		fx.Invoke(config.InitLog, config.InitDNS),
		fx.Populate(android.Client),
	)
	err := android.App.Start(ctx)
	if err != nil {
		log.Error(err)
	}
	err = android.Client.Run()
	if err != nil {
		log.Error(err)
	}
}

func (android *Android) Shutdown() {
	android.Client.Shutdown()
	err := android.App.Stop(context.Background())
	if err != nil {
		log.Error(err)
	}
}

func (android *Android) ServeDNS() {

}

func InputPacket(data []byte) {
	lwipStack.Write(data)
}

func StartSocks(packetFlow PacketFlow, proxyHost string, proxyPort int) {
	if packetFlow != nil {
		lwipStack = core.NewLWIPStack()
		core.RegisterTCPConnHandler(socks.NewTCPHandler(proxyHost, uint16(proxyPort)))
		core.RegisterUDPConnHandler(socks.NewUDPHandler(proxyHost, uint16(proxyPort), 2*time.Minute))
		core.RegisterOutputFn(func(data []byte) (int, error) {
			packetFlow.WritePacket(data)
			return len(data), nil
		})
	}
}
