package main

import (
	"github.com/iyouport-org/relaybaton"
	"github.com/iyouport-org/relaybaton/config"
	"github.com/iyouport-org/relaybaton/dns"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net"
	"net/http"
	"os"
	"strconv"
)

func main() {
	err := os.Setenv("GODEBUG", os.Getenv("GODEBUG")+",tls13=1,netdns=go")
	if err != nil {
		log.Fatal(err)
		return
	}
	v := viper.New()
	v.SetConfigName("config")
	v.AddConfigPath(".")
	if err := v.ReadInConfig(); err != nil {
		log.Error(err)
		return
	}
	var conf config.MainConfig
	err = v.Unmarshal(&conf)
	if err != nil {
		log.Error(err)
		return
	}
	err = conf.Init()
	if err != nil {
		log.Error(err)
		return
	}
	log.SetOutput(conf.LogFileFile)
	log.SetLevel(log.TraceLevel)
	log.SetFormatter(relaybaton.XMLFormatter{})
	log.SetReportCaller(true)

	switch conf.DNS.Type {
	case "dot":
		net.DefaultResolver = dns.NewDoTResolverFactory(net.Dialer{}, conf.DNS.Server, conf.DNS.Addr, false).GetResolver()
	case "doh":
		factory, err := dns.NewDoHResolverFactory(net.Dialer{}, 11111, conf.DNS.Server, conf.DNS.Addr, false)
		if err != nil {
			log.Error(err)
			return
		}
		net.DefaultResolver = factory.GetResolver()
	}

	switch os.Args[1] {
	case "client":
		for {
			client, err := relaybaton.NewClient(conf)
			if err != nil {
				log.Error(err)
				continue
			}
			client.Run()
		}
	case "server":
		handler := relaybaton.Handler{
			Conf: conf,
		}
		log.Error(http.ListenAndServe(":"+strconv.Itoa(conf.Server.Port), handler))
	}
}
