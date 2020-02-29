package config

import (
	log "github.com/sirupsen/logrus"
	"net"
)

type DNSType string

const (
	DNSTypeDefault DNSType = "default"
	DNSTypeDoT     DNSType = "dot"
	DNSTypeDoH     DNSType = "doh"
)

type dnsTOML struct {
	Type         string `mapstructure:"type"`
	Server       string `mapstructure:"server"`
	Addr         string `mapstructure:"addr"`
	LocalResolve bool   `mapstructure:"local_resolve"`
}

type dnsGo struct {
	Type         DNSType
	Server       string
	Addr         net.Addr
	LocalResolve bool
}

func (dnst *dnsTOML) Init() (dnsg *dnsGo, err error) {
	dnsg = &dnsGo{
		Server:       dnst.Server,
		LocalResolve: dnst.LocalResolve,
	}
	switch dnst.Type {
	case "dot":
		dnsg.Type = DNSTypeDoT
		dnsg.Addr, err = net.ResolveTCPAddr("tcp", dnst.Addr)
		if err != nil {
			log.WithField("dns.addr", dnst.Addr).Error(err)
			return nil, err
		}
	case "doh":
		dnsg.Type = DNSTypeDoH
		dnsg.Addr, err = net.ResolveTCPAddr("ip", dnst.Addr)
		if err != nil {
			log.WithField("dns.addr", dnst.Addr).Error(err)
			return nil, err
		}
	default:
		dnsg.Type = DNSTypeDefault
	}
	return dnsg, nil
}
