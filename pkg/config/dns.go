package config

import (
	"errors"
	"net"

	"github.com/asaskevich/govalidator"
	log "github.com/sirupsen/logrus"
)

type DNSType string

const (
	DNSTypeDefault DNSType = "default"
	DNSTypeDoT     DNSType = "dot"
	DNSTypeDoH     DNSType = "doh"
)

type DNSToml struct {
	Type   string `mapstructure:"type" toml:"type" validate:"oneof='default' 'dot' 'doh',required"`
	Server string `mapstructure:"server" toml:"server" validate:"omitempty,required,hostname|hostname_rfc1123|fqdn,required"`
	Addr   string `mapstructure:"addr" toml:"addr" validate:"omitempty,required,ip|ip_addr|tcp_addr|udp_addr,required"`
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
		if !govalidator.IsIP(dnst.Addr) {
			err = errors.New("wrong IP address")
			log.WithField("dns.addr", dnst.Addr).Error(err)
			return nil, err
		}
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
