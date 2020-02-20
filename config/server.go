package config

import log "github.com/sirupsen/logrus"

type serverConfig struct {
	Port    int    `mapstructure:"port"`
	Pretend string `mapstructure:"pretend"`
}

func (sc *serverConfig) Init() error {
	err := sc.validate()
	if err != nil {
		log.Debug(err)
		return err
	}
	return nil
}

func (sc *serverConfig) validate() error {
	return nil
}
