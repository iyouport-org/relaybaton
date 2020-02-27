package config

import (
	"errors"
	log "github.com/sirupsen/logrus"
)

type clientTOML struct {
	ID       string `mapstructure:"id"`
	Server   string `mapstructure:"server"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	ESNI     bool   `mapstructure:"bool"`
}

type ClientGo struct {
	ID       string
	Server   string
	Username string
	Password string
	ESNI     bool
}

func (ct *clientTOML) Init() (cg *ClientGo, err error) {
	if ct.ID == "default" {
		err = errors.New("client id cannot be 'default'")
		log.Error(err)
		return nil, err
	}
	return &ClientGo{
		ID:       ct.ID,
		Server:   ct.Server,
		Username: ct.Username,
		Password: ct.Password,
		ESNI:     ct.ESNI,
	}, nil
}
