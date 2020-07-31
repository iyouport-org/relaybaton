package config

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net"
	"relaybaton/pkg/dns"
	"relaybaton/pkg/log"
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
		logrus.Error(err)
		return nil, err
	}
	cg.DNS, err = mc.DNS.Init()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return cg, nil
}

func NewConf() *ConfigGo {
	v := viper.New()
	if viper.GetString("config") != "" {
		logrus.Debug(viper.GetString("config"))
		v.SetConfigFile(viper.GetString("config"))
	} else {
		v.SetConfigName("config")
		v.SetConfigType("toml")
		v.AddConfigPath(".")
	}
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		logrus.Panic(err)
	}
	logrus.Debug(v.ConfigFileUsed())
	var confTOML ConfigTOML
	err := v.Unmarshal(&confTOML)
	if err != nil {
		logrus.Panic(err)
	}
	conf, err := confTOML.Init()
	if err != nil {
		logrus.Panic(err)
	}

	return conf
}

func NewConfClient() (conf *ConfigGo, err error) {
	conf = NewConf()
	conf.Clients, err = conf.toml.Clients.Init()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	conf.Routes, err = conf.toml.Routes.Init()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	for _, v := range conf.Routes.Route {
		if conf.Clients.Client[v.Target] == nil && v.Target != "default" {
			err = errors.New(fmt.Sprintf("target %s do not exist", v.Target))
			logrus.WithField("routes.route.target", v.Target).Error(err)
			return nil, err
		}
	}
	return conf, nil
}

func NewConfServer() (conf *ConfigGo, err error) {
	conf = NewConf()
	conf.Server, err = conf.toml.Server.Init()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	conf.DB, err = conf.toml.DB.Init()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return conf, nil
}

func InitLog(conf *ConfigGo) {
	logrus.SetFormatter(log.XMLFormatter{})
	logrus.SetOutput(conf.Log.File)
	logrus.SetLevel(conf.Log.Level)
	if conf.DB == nil {
		db, err := gorm.Open("sqlite3", "log.db")
		if err != nil {
			logrus.Error(err)
			return
		}
		db.AutoMigrate(&log.Record{})
		logrus.AddHook(log.NewSQLiteHook(db))
	} else {
		conf.DB.DB.AutoMigrate(&log.Record{})
		logrus.AddHook(log.NewSQLiteHook(conf.DB.DB))
	}
}

func InitDNS(conf *ConfigGo) {
	switch conf.DNS.Type {
	case DNSTypeDoT:
		factory := dns.NewDoTResolverFactory(net.Dialer{}, conf.DNS.Server, conf.DNS.Addr, false)
		net.DefaultResolver = factory.GetResolver()
	case DNSTypeDoH:
		factory, err := dns.NewDoHResolverFactory(net.Dialer{}, 1083, conf.DNS.Server, conf.DNS.Addr, false)
		if err != nil {
			logrus.Error(err)
			return
		}
		net.DefaultResolver = factory.GetResolver()
	}
}
