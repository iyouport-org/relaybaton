package config

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net"
	"relaybaton-dev/pkg/dns"
	formatter "relaybaton-dev/pkg/log"
)

// ConfigTOML is the struct mapped from the configuration file
type ConfigTOML struct {
	Log     logTOML     `mapstructure:"log"`
	DNS     dnsTOML     `mapstructure:"dns"`
	Clients clientsTOML `mapstructure:"clients"`
	Routes  routesTOML  `mapstructure:"routes"`
	Server  serverTOML  `mapstructure:"server"`
	DB      dbTOML      `mapstructure:"db"`
}

type ConfigGo struct {
	toml    *ConfigTOML
	Log     *logGo     //client,server
	DNS     *dnsGo     //client,server
	Clients *clientsGo //client
	Routes  *routesGo  //client
	Server  *serverGo  //server
	DB      *dbGo      //server
}

func (mc *ConfigTOML) Init() (cg *ConfigGo, err error) {
	cg = &ConfigGo{}
	cg.toml = mc
	cg.Log, err = mc.Log.Init()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	cg.DNS, err = mc.DNS.Init()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return cg, nil
}

func NewConf() *ConfigGo {
	v := viper.New()
	if viper.GetString("config") != "" {
		log.Debug(viper.GetString("config"))
		v.SetConfigFile(viper.GetString("config"))
	} else {
		v.SetConfigName("config")
		v.SetConfigType("toml")
		v.AddConfigPath(".")
	}
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		log.Panic(err)
	}
	log.Debug(v.ConfigFileUsed())
	var confTOML ConfigTOML
	err := v.Unmarshal(&confTOML)
	if err != nil {
		log.Panic(err)
	}
	conf, err := confTOML.Init()
	if err != nil {
		log.Panic(err)
	}

	return conf
}

func NewConfClient() (conf *ConfigGo, err error) {
	conf = NewConf()
	conf.Clients, err = conf.toml.Clients.Init()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	conf.Routes, err = conf.toml.Routes.Init()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	for _, v := range conf.Routes.Route {
		if conf.Clients.Client[v.Target] == nil && v.Target != "default" {
			err = errors.New(fmt.Sprintf("target %s do not exist", v.Target))
			log.WithField("routes.route.target", v.Target).Error(err)
			return nil, err
		}
	}
	return conf, nil
}

func NewConfServer() (conf *ConfigGo, err error) {
	conf = NewConf()
	conf.Server, err = conf.toml.Server.Init()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	conf.DB, err = conf.toml.DB.Init()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return conf, nil
}

func InitLog(conf *ConfigGo) {
	log.SetFormatter(formatter.XMLFormatter{})
	log.SetReportCaller(true)
	log.SetOutput(conf.Log.File)
	log.SetLevel(conf.Log.Level)
}

func InitDNS(conf *ConfigGo) {
	switch conf.DNS.Type {
	case DNSTypeDoT:
		factory := dns.NewDoTResolverFactory(net.Dialer{}, conf.DNS.Server, conf.DNS.Addr, false)
		net.DefaultResolver = factory.GetResolver()
	case DNSTypeDoH:
		factory, err := dns.NewDoHResolverFactory(net.Dialer{}, 1083, conf.DNS.Server, conf.DNS.Addr, false)
		if err != nil {
			log.Error(err)
			return
		}
		net.DefaultResolver = factory.GetResolver()
	}
}
