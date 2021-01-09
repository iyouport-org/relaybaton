package relaybaton_mobile

import (
	"bytes"
	"context"
	"os"
	"runtime/debug"

	tun2socks "github.com/eycorsican/go-tun2socks/core"
	"github.com/eycorsican/go-tun2socks/proxy/socks"
	"github.com/go-playground/validator/v10"
	"github.com/iyouport-org/relaybaton/pkg/config"
	"github.com/iyouport-org/relaybaton/pkg/core"
	"github.com/panjf2000/gnet/pool/goroutine"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var lwipStack tun2socks.LWIPStack

type RelaybatonAndroid struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	conf       *config.ConfigGo
	client     *core.Client
	app        *fx.App
}

type AndroidError struct {
	error
}

func (err *AndroidError) Error() string {
	return err.error.Error()
}

func NewAndroid(conf string) (*RelaybatonAndroid, error) {
	core.IsMobile = true
	debug.SetMaxThreads(1 << 20)
	err := os.Setenv("GODEBUG", os.Getenv("GODEBUG")+",tls13=1,netdns=go")
	if err != nil {
		log.Error(err)
		return nil, &AndroidError{err}
	}
	log.SetReportCaller(true)
	log.SetLevel(log.TraceLevel)

	ra := &RelaybatonAndroid{}
	validate := validator.New()
	v := viper.New()
	v.SetConfigType("toml")
	err = v.ReadConfig(bytes.NewBufferString(conf))
	if err != nil {
		log.Error(err)
		return nil, &AndroidError{err}
	}
	var confTOML config.ConfigTOML
	err = v.Unmarshal(&confTOML)
	if err != nil {
		log.Error(err)
		return nil, &AndroidError{err}
	}
	ra.conf, err = confTOML.Init()
	if err != nil {
		log.Error(err)
		return nil, &AndroidError{err}
	}
	err = validate.Struct(confTOML.Client)
	if err != nil {
		log.Error(err)
		return nil, &AndroidError{err}
	}
	ra.conf.Client, err = confTOML.Client.Init()
	if err != nil {
		log.Error(err)
		return nil, &AndroidError{err}
	}
	return ra, nil
}

type PacketFlow interface {
	WritePacket(packet []byte)
}

func (android *RelaybatonAndroid) Run() error {
	android.ctx, android.cancelFunc = context.WithCancel(context.Background())
	defer android.cancelFunc()
	android.app = fx.New(
		fx.Provide(
			func() *config.ConfigGo {
				return android.conf
			},
			goroutine.Default,
			core.NewRouter,
			core.NewClient,
		),
		fx.Logger(log.StandardLogger()),
		fx.Invoke(config.InitLog, config.InitDNS),
		fx.Populate(&android.client),
	)
	err := android.app.Start(android.ctx)
	if err != nil {
		log.Error(err)
		return &AndroidError{err}
	}
	err = android.client.Run()
	if err != nil {
		log.Error(err)
		return &AndroidError{err}
	}
	return &AndroidError{err}
}

func (android *RelaybatonAndroid) Save(clientServer string, clientUser string, clientPassword string, clientProxyAll bool, DNSType string, DNSServer string, DNSAddr string, logLevel string) error {
	confTOML := config.ConfigTOML{
		Log: &config.LogTOML{
			File:  "/data/data/org.iyouport.relaybaton_android/files/log.xml",
			Level: logLevel,
		},
		DNS: &config.DNSToml{
			Type:   DNSType,
			Server: DNSServer,
			Addr:   DNSAddr,
		},
		Client: &config.ClientTOML{
			Port:      1080,
			HTTPPort:  1088,
			RedirPort: 1090,
			Server:    clientServer,
			Username:  clientUser,
			Password:  clientPassword,
			ProxyAll:  clientProxyAll,
		},
	}
	conf, err := confTOML.Init()
	if err != nil {
		log.Error(err)
		return &AndroidError{err}
	}
	conf.Client, err = confTOML.Client.Init()
	if err != nil {
		log.Error(err)
		return &AndroidError{err}
	}
	err = conf.SaveClient("/data/data/org.iyouport.relaybaton_android/files/config.toml")
	if err != nil {
		log.Error(err)
		return &AndroidError{err}
	}
	android.conf = conf
	return nil
}

func (android *RelaybatonAndroid) Shutdown() error {
	android.cancelFunc()
	err := lwipStack.Close()
	if err != nil {
		log.Error(err)
		return &AndroidError{err}
	}
	return &AndroidError{android.app.Stop(android.ctx)}
}

func (android *RelaybatonAndroid) GetClientServer() string {
	return android.conf.Client.Server
}

func (android *RelaybatonAndroid) GetClientUsername() string {
	return android.conf.Client.Username
}

func (android *RelaybatonAndroid) GetClientPassword() string {
	return android.conf.Client.Password
}

func (android *RelaybatonAndroid) GetClientProxyAll() bool {
	return android.conf.Client.ProxyAll
}

func (android *RelaybatonAndroid) GetDNSType() string {
	return string(android.conf.DNS.Type)
}

func (android *RelaybatonAndroid) GetDNSServer() string {
	return android.conf.DNS.Server
}

func (android *RelaybatonAndroid) GetDNSAddr() string {
	return android.conf.DNS.Addr.String()
}

func (android *RelaybatonAndroid) GetLogLevel() string {
	return android.conf.Log.Level.String()
}

func (android *RelaybatonAndroid) InputPacket(data []byte) {
	lwipStack.Write(data)
}

func (android *RelaybatonAndroid) StartSocks(packetFlow PacketFlow, proxyHost string, proxyPort int) {
	if packetFlow != nil {
		lwipStack = tun2socks.NewLWIPStack()
		tun2socks.RegisterTCPConnHandler(socks.NewTCPHandler(proxyHost, uint16(proxyPort)))
		tun2socks.RegisterUDPConnHandler(NewUDPHandler())
		tun2socks.RegisterOutputFn(func(data []byte) (int, error) {
			packetFlow.WritePacket(data)
			return len(data), nil
		})
	}
}
