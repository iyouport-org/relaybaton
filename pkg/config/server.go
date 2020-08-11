package config

import (
	log "github.com/sirupsen/logrus"
	"net/url"
)

type ServerTOML struct {
	Port    int    `mapstructure:"port" toml:"port" validate:"numeric,gte=0,lte=65535,required"`
	Pretend string `mapstructure:"pretend" toml:"pretend" validate:"required"`
}

type serverGo struct {
	Port     uint16
	Pretend  *url.URL
}

func (st *ServerTOML) Init() (sg *serverGo, err error) {
	sg = &serverGo{
		Port: uint16(st.Port),
	}
	sg.Pretend, err = url.Parse(st.Pretend)
	if err != nil {
		log.WithField("server.pretend", st.Pretend).Error(err)
		return nil, err
	}
	return sg, nil
}
