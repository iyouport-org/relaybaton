package config

import (
	log "github.com/sirupsen/logrus"
	"os"
)

// MainConfig is the struct mapped from the configuration file
type MainConfig struct {
	LogFileString string        `mapstructure:"log_file"`
	Client        clientConfig  `mapstructure:"client"`
	Server        serverConfig  `mapstructure:"server"`
	DNS           dnsConfig     `mapstructure:"dns"`
	Routing       routingConfig `mapstructure:"routing"`
	DB            dbConfig      `mapstructure:"db"`
	LogFileFile   *os.File
}

func (mc *MainConfig) Init() (err error) {
	err = mc.validate()
	if err != nil {
		log.Error(err)
		return err
	}
	err = mc.Client.Init()
	if err != nil {
		log.Error(err)
		return err
	}
	err = mc.Server.Init()
	if err != nil {
		log.Error(err)
		return err
	}
	err = mc.DNS.Init()
	if err != nil {
		log.Error(err)
		return err
	}
	err = mc.Routing.Init()
	if err != nil {
		log.Error(err)
		return err
	}
	err = mc.DB.Init()
	if err != nil {
		log.Error(err)
		return err
	}
	mc.LogFileFile, err = os.OpenFile(mc.LogFileString, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (mc *MainConfig) validate() (err error) {
	_, err = os.Stat(mc.LogFileString)
	if err != nil && !os.IsNotExist(err) {
		log.Error(err)
		return err
	}
	err = mc.Client.validate()
	if err != nil {
		log.Error(err)
		return err
	}
	err = mc.Server.validate()
	if err != nil {
		log.Error(err)
		return err
	}
	err = mc.DNS.validate()
	if err != nil {
		log.Error(err)
		return err
	}
	err = mc.Routing.validate()
	if err != nil {
		log.Error(err)
		return err
	}
	err = mc.DB.validate()
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}
