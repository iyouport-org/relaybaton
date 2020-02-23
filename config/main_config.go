package config

import (
	log "github.com/sirupsen/logrus"
)

// ConfigTOML is the struct mapped from the configuration file
type ConfigTOML struct {
	Log     logTOML     `mapstructure:"log"`
	Client  clientTOML  `mapstructure:"client"`
	Server  serverTOML  `mapstructure:"server"`
	DNS     dnsTOML     `mapstructure:"dns"`
	Routing routingTOML `mapstructure:"routing"`
	DB      dbTOML      `mapstructure:"db"`
}

type ConfigGo struct {
	Log     *logGo     //client,server
	Client  *clientGo  //client
	Server  *serverGo  //server
	DNS     *dnsGo     //client,server
	Routing *routingGo //client
	DB      *dbGo      //server
}

func (mc *ConfigTOML) Init() (cg *ConfigGo, err error) {
	cg = &ConfigGo{}
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

func (mc *ConfigTOML) InitClient(cg *ConfigGo) error {
	var err error
	cg.Client, err = mc.Client.Init()
	if err != nil {
		log.Error(err)
		return err
	}
	cg.Routing, err = mc.Routing.Init()
	if err != nil {
		log.Error(err)
		return err
	}
	if cg.Routing.Type != RoutingTypeDefault && !cg.DNS.LocalResolve {
		log.Warn("dns.local_resolve should be set true for routing")
	}
	if cg.Routing.Type != RoutingTypeDefault && cg.DNS.Type == DNSTypeDefault {
		log.Warn("Secure DNS should be set for routing")
	}
	return nil
}

func (mc *ConfigTOML) InitServer(cg *ConfigGo) error {
	var err error
	cg.Server, err = mc.Server.Init()
	if err != nil {
		log.Error(err)
		return err
	}
	cg.DB, err = mc.DB.Init()
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}
