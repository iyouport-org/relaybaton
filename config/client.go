package config

import log "github.com/sirupsen/logrus"

type clientConfig struct {
	Server   string `mapstructure:"server"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

func (cc *clientConfig) Init() error {
	err := cc.validate()
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (cc *clientConfig) validate() error {
	return nil
}
