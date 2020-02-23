package main

import (
	"fmt"
	"github.com/iyouport-org/relaybaton"
	"github.com/iyouport-org/relaybaton/config"
	"github.com/iyouport-org/relaybaton/dns"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net"
	"net/http"
	"os"
)

func main() {
	err := os.Setenv("GODEBUG", os.Getenv("GODEBUG")+",tls13=1,netdns=go")
	if err != nil {
		log.Fatal(err)
		return
	}

	log.SetFormatter(relaybaton.XMLFormatter{})
	log.SetReportCaller(true)

	v := viper.New()
	v.SetConfigName("config")
	v.AddConfigPath(".")
	if err := v.ReadInConfig(); err != nil {
		log.Error(err)
		return
	}
	var confTOML config.ConfigTOML
	err = v.Unmarshal(&confTOML)
	if err != nil {
		log.Error(err)
		return
	}
	conf, err := confTOML.Init()
	if err != nil {
		log.Error(err)
		return
	}

	log.SetOutput(conf.Log.File)
	log.SetLevel(conf.Log.Level)

	switch conf.DNS.Type {
	case config.DNSTypeDoT:
		net.DefaultResolver = dns.NewDoTResolverFactory(net.Dialer{}, conf.DNS.Server, conf.DNS.Addr, false).GetResolver()
	case config.DNSTypeDoH:
		factory, err := dns.NewDoHResolverFactory(net.Dialer{}, 11111, conf.DNS.Server, conf.DNS.Addr, false)
		if err != nil {
			log.Error(err)
			return
		}
		net.DefaultResolver = factory.GetResolver()
	}

	switch os.Args[1] {
	case "client":
		err = confTOML.InitClient(conf)
		if err != nil {
			log.Error(err)
			return
		}
		for {
			client, err := relaybaton.NewClient(conf)
			if err != nil {
				log.Error(err)
				continue
			}
			client.Run()
		}
	case "server":
		err = confTOML.InitServer(conf)
		if err != nil {
			log.Error(err)
			return
		}
		handler := relaybaton.Handler{
			Conf: conf,
		}
		log.Error(http.ListenAndServe(fmt.Sprintf(":%d", conf.Server.Port), handler))
	}
}
