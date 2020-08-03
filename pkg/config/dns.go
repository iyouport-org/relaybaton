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

type DNSToml struct {
	Type   string `mapstructure:"type" toml:"type" validate:"required"`
	Server string `mapstructure:"server" toml:"server" validate:"required,hostname"`
	Addr   string `mapstructure:"addr" toml:"addr" validate:"required,ip|tcp_addr"`
}

type DNSGo struct {
	Type   DNSType
	Server string
	Addr   net.Addr
}

func (dnst *DNSToml) Init() (dnsg *DNSGo, err error) {
	dnsg = &DNSGo{
		Server: dnst.Server,
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
		dnsg.Addr, err = net.ResolveIPAddr("ip", dnst.Addr)
		if err != nil {
			log.WithField("dns.addr", dnst.Addr).Error(err)
			return nil, err
		}
	default:
		dnsg.Type = DNSTypeDefault
	}
	return dnsg, nil
}
