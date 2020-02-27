package config

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
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
	Log     *logGo     //client,server
	DNS     *dnsGo     //client,server
	Clients *clientsGo //client
	Routes  *routesGo  //client
	Server  *serverGo  //server
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
	cg.Clients, err = mc.Clients.Init()
	if err != nil {
		log.Error(err)
		return err
	}
	cg.Routes, err = mc.Routes.Init()
	if err != nil {
		log.Error(err)
		return err
	}
	for _, v := range cg.Routes.Route {
		if cg.Clients.Client[v.Target] == nil && v.Target != "default" {
			err = errors.New(fmt.Sprintf("target %s do not exist", v.Target))
			log.WithField("routes.route.target", v.Target).Error(err)
			return err
		}
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
