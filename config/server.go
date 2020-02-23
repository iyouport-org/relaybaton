package config

import (
	log "github.com/sirupsen/logrus"
	"net/url"
)

type serverTOML struct {
	Port    int    `mapstructure:"port"`
	Pretend string `mapstructure:"pretend"`
}

type serverGo struct {
	Port    uint16
	Pretend *url.URL
}

func (st *serverTOML) Init() (sg *serverGo, err error) {
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
