package config

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
)

type clientsTOML struct {
	Port   int          `mapstructure:"port"`
	Client []clientTOML `mapstructure:"client"`
}

type clientsGo struct {
	Port   uint16
	Client map[string]*ClientGo
}

func (ct *clientsTOML) Init() (cg *clientsGo, err error) {
	cg = &clientsGo{
		Port:   uint16(ct.Port),
		Client: map[string]*ClientGo{},
	}
	for _, v := range ct.Client {
		client, err := v.Init()
		if err != nil {
			log.WithField("clients.client.id", v.ID).Error(err)
			return nil, err
		}
		if cg.Client[client.ID] == nil {
			cg.Client[client.ID] = client
		} else {
			err = errors.New(fmt.Sprintf("duplicated client id: %s", client.ID))
			log.WithField("clients.client.id", client.ID).Error(err)
			return nil, err
		}
	}
	return cg, nil
}
