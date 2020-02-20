package config

import log "github.com/sirupsen/logrus"

type dnsConfig struct {
	Type         string `mapstructure:"type"`
	Server       string `mapstructure:"server"`
	Addr         string `mapstructure:"addr"`
	LocalResolve bool   `mapstructure:"local_resolve"`
}

func (dnsc dnsConfig) Init() error {
	err := dnsc.validate()
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (dnsc dnsConfig) validate() error {
	return nil
}
