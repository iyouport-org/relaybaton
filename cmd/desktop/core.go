package main

import (
	"C"
	"context"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/iyouport-org/relaybaton/internal/cmd/relaybaton"
	"github.com/iyouport-org/relaybaton/pkg/config"
	"github.com/iyouport-org/relaybaton/pkg/core"
	"github.com/panjf2000/gnet/pool/goroutine"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var app *fx.App
var cancel context.CancelFunc
var ctx context.Context

//export Init
func Init() {
	debug.SetMaxThreads(1 << 20)
	err := os.Setenv("GODEBUG", os.Getenv("GODEBUG")+",tls13=1,netdns=go")
	if err != nil {
		log.Fatal(err)
		return
	}
	err = viper.BindPFlag("config", relaybaton.RootCmd.PersistentFlags().Lookup("config"))
	if err != nil {
		log.Error(err)
	}
	log.SetReportCaller(true)
	log.SetLevel(log.TraceLevel)
}

//export OpenConfFile
func OpenConfFile(filePath *C.char) (clientPort C.int, clientHTTPPort C.int, clientRedirPort C.int, clientServer *C.char, clientUsername *C.char, clientPassword *C.char, clientProxyAll C.int,
	DNSType *C.char, DNSServer *C.char, DNSAddr *C.char,
	logFile *C.char, logLevel *C.char, ret *C.char) {
	clientPort = C.int(0)
	clientHTTPPort = C.int(0)
	clientRedirPort = C.int(0)
	clientServer = C.CString("")
	clientUsername = C.CString("")
	clientPassword = C.CString("")
	clientProxyAll = C.int(0)
	DNSType = C.CString("")
	DNSServer = C.CString("")
	DNSAddr = C.CString("")
	logFile = C.CString("")
	logLevel = C.CString("")
	ret = C.CString("")

	path := C.GoString(filePath)
	validate := validator.New()
	v := viper.New()
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(filepath.Base(path)))
	v.SetConfigName(name)
	v.SetConfigType("toml")
	v.AddConfigPath(filepath.Dir(path))
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		ret = C.CString(err.Error())
		return
	}
	var confTOML config.ConfigTOML
	err := v.Unmarshal(&confTOML)
	if err != nil {
		ret = C.CString(err.Error())
		return
	}
	err = validate.Struct(confTOML)
	if err != nil {
		ret = C.CString(err.Error())
		return
	}
	err = validate.Struct(confTOML.Client)
	if err != nil {
		ret = C.CString(err.Error())
		return
	}
	clientPort = C.int(confTOML.Client.Port)
	clientHTTPPort = C.int(confTOML.Client.HTTPPort)
	clientRedirPort = C.int(confTOML.Client.RedirPort)
	clientServer = C.CString(confTOML.Client.Server)
	clientUsername = C.CString(confTOML.Client.Username)
	clientPassword = C.CString(confTOML.Client.Password)
	if confTOML.Client.ProxyAll {
		clientProxyAll = C.int(1)
	} else {
		clientProxyAll = C.int(0)
	}
	DNSType = C.CString(confTOML.DNS.Type)
	DNSServer = C.CString(confTOML.DNS.Server)
	DNSAddr = C.CString(confTOML.DNS.Addr)
	logFile = C.CString(confTOML.Log.File)
	logLevel = C.CString(confTOML.Log.Level)
	return
}

//export SaveConfFile
func SaveConfFile(filePath *C.char,
	clientPort C.int, clientHTTPPort C.int, clientRedirPort C.int, clientServer *C.char, clientUsername *C.char, clientPassword *C.char, clientProxyAll C.int,
	DNSType *C.char, DNSServer *C.char, DNSAddr *C.char,
	logFile *C.char, logLevel *C.char) *C.char {
	var proxyAll bool
	if int(clientProxyAll) == 0 {
		proxyAll = false
	} else {
		proxyAll = true
	}
	validate := validator.New()
	conf := config.ConfigTOML{
		Log: &config.LogTOML{
			File:  C.GoString(logFile),
			Level: C.GoString(logLevel),
		},
		DNS: &config.DNSToml{
			Type:   C.GoString(DNSType),
			Server: C.GoString(DNSServer),
			Addr:   C.GoString(DNSAddr),
		},
		Client: &config.ClientTOML{
			Port:      int(clientPort),
			HTTPPort:  int(clientHTTPPort),
			RedirPort: int(clientRedirPort),
			Server:    C.GoString(clientServer),
			Username:  C.GoString(clientUsername),
			Password:  C.GoString(clientPassword),
			ProxyAll:  proxyAll,
		},
	}
	err := validate.Struct(conf)
	if err != nil {
		return C.CString(err.Error())
	}
	err = validate.Struct(conf.Client)
	if err != nil {
		return C.CString(err.Error())
	}
	confGo, err := conf.Init()
	if err != nil {
		return C.CString(err.Error())
	}
	confGo.Client, err = conf.Client.Init()
	if err != nil {
		return C.CString(err.Error())
	}
	err = confGo.SaveClient(C.GoString(filePath))
	if err != nil {
		return C.CString(err.Error())
	}
	return C.CString("")
}

//export Validate
func Validate(clientPort C.int, clientHTTPPort C.int, clientRedirPort C.int, clientServer *C.char, clientUsername *C.char, clientPassword *C.char, clientProxyAll C.int,
	DNSType *C.char, DNSServer *C.char, DNSAddr *C.char,
	logFile *C.char, logLevel *C.char) *C.char {
	var proxyAll bool
	if int(clientProxyAll) == 0 {
		proxyAll = false
	} else {
		proxyAll = true
	}
	validate := validator.New()
	conf := config.ConfigTOML{
		Log: &config.LogTOML{
			File:  C.GoString(logFile),
			Level: C.GoString(logLevel),
		},
		DNS: &config.DNSToml{
			Type:   C.GoString(DNSType),
			Server: C.GoString(DNSServer),
			Addr:   C.GoString(DNSAddr),
		},
		Client: &config.ClientTOML{
			Port:      int(clientPort),
			HTTPPort:  int(clientHTTPPort),
			RedirPort: int(clientRedirPort),
			Server:    C.GoString(clientServer),
			Username:  C.GoString(clientUsername),
			Password:  C.GoString(clientPassword),
			ProxyAll:  proxyAll,
		},
	}
	err := validate.Struct(conf)
	if err != nil {
		return C.CString(err.Error())
	}
	err = validate.Struct(conf.Client)
	if err != nil {
		return C.CString(err.Error())
	}
	return C.CString("")
}

//export Run
func Run(clientPort C.int, clientHTTPPort C.int, clientRedirPort C.int, clientServer *C.char, clientUsername *C.char, clientPassword *C.char, clientProxyAll C.int,
	DNSType *C.char, DNSServer *C.char, DNSAddr *C.char,
	logFile *C.char, logLevel *C.char) *C.char {
	var proxyAll bool
	if int(clientProxyAll) == 0 {
		proxyAll = false
	} else {
		proxyAll = true
	}
	ctx, cancel = context.WithCancel(context.Background())
	var client *core.Client
	app = fx.New(
		fx.Provide(
			core.NewClient,
			func() (*config.ConfigGo, error) {
				conf := config.ConfigTOML{
					Log: &config.LogTOML{
						File:  C.GoString(logFile),
						Level: C.GoString(logLevel),
					},
					DNS: &config.DNSToml{
						Type:   C.GoString(DNSType),
						Server: C.GoString(DNSServer),
						Addr:   C.GoString(DNSAddr),
					},
					Client: &config.ClientTOML{
						Port:      int(clientPort),
						HTTPPort:  int(clientHTTPPort),
						RedirPort: int(clientRedirPort),
						Server:    C.GoString(clientServer),
						Username:  C.GoString(clientUsername),
						Password:  C.GoString(clientPassword),
						ProxyAll:  proxyAll,
					},
				}
				confGo, err := conf.Init()
				if err != nil {
					return nil, err
				}
				err = confGo.InitClient()
				if err != nil {
					return nil, err
				}
				return confGo, nil
			},
			goroutine.Default,
			core.NewRouter,
		),
		fx.Logger(log.StandardLogger()),
		fx.Invoke(config.InitLog, config.InitDNS),
		fx.Populate(&client),
	)
	err := app.Start(ctx)
	if err != nil {
		return C.CString(err.Error())
	}
	go func() {
		defer func() {
			cancel()
			err := app.Stop(ctx)
			if err != nil {
				log.Error(err)
			}
		}()
		err = client.Run()
		if err != nil {
			log.Error(err)
		}
	}()
	return C.CString("")
}

//export Stop
func Stop() {
	cancel()
	err := app.Stop(ctx)
	if err != nil {
		log.Error(err)
	}
}

func main() {}
