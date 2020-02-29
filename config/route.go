package config

import (
	"errors"
	"github.com/oschwald/geoip2-golang"
	log "github.com/sirupsen/logrus"
	"net"
	"regexp"
	"strings"
)

type routeType string

const (
	RouteTypeDefault    routeType = "default"
	RouteTypeGeoIP      routeType = "geoip"
	RouteTypeDomain     routeType = "domain"
	RouteTypeIPv4       routeType = "ipv4"
	RouteTypeIPv6       routeType = "ipv6"
	RouteTypeIPv4Subnet routeType = "ipv4subnet"
	RouteTypeIPv6Subnet routeType = "ipv6subnet"
)

type routeTOML struct {
	Type   string `mapstructure:"type"`
	Cond   string `mapstructure:"cond"`
	Target string `mapstructure:"target"`
}

type routeGo struct {
	Type         routeType
	GeoIPDB      *geoip2.Reader
	CondGeoIP    []string
	CondDomain   *regexp.Regexp
	CondIP       net.IP
	CondIPSubnet *net.IPNet
	Target       string
}

func (rt *routeTOML) Init(GeoIPDB *geoip2.Reader) (rg *routeGo, err error) {
	rg = &routeGo{
		Target: rt.Target,
	}
	switch rt.Type {
	case "geoip":
		rg.Type = RouteTypeGeoIP
		rg.GeoIPDB = GeoIPDB
		rg.CondGeoIP = strings.Split(rt.Cond, ",")
	case "domain":
		rg.Type = RouteTypeDomain
		rg.CondDomain, err = regexp.Compile(rt.Cond)
		if err != nil {
			log.WithField("routes.route.cond", rt.Cond).Error(err)
			return nil, err
		}
	case "ipv4":
		rg.Type = RouteTypeIPv4
		rg.CondIP = net.ParseIP(rt.Cond)
		if rg.CondIP == nil {
			err = errors.New("unknown IP address: " + rt.Cond)
			log.WithField("routes.route.cond", rt.Cond).Error(err)
			return nil, err
		}
		if rg.CondIP.To4() == nil {
			err = errors.New("unknown IPv4 address: " + rt.Cond)
			log.WithField("routes.route.cond", rt.Cond).Error(err)
			return nil, err
		}
	case "ipv6":
		rg.Type = RouteTypeIPv6
		rg.CondIP = net.ParseIP(rt.Cond)
		if rg.CondIP == nil {
			err = errors.New("unknown IP address: " + rt.Cond)
			log.WithField("routes.route.cond", rt.Cond).Error(err)
			return nil, err
		}
		if rg.CondIP.To16() == nil {
			err = errors.New("unknown IPv6 address: " + rt.Cond)
			log.WithField("routes.route.cond", rt.Cond).Error(err)
			return nil, err
		}
	case "ipv4subnet", "ipv6subnet":
		rg.Type = RouteTypeIPv4Subnet
		_, rg.CondIPSubnet, err = net.ParseCIDR(rt.Cond)
		if err != nil {
			log.WithField("routes.route.cond", rt.Cond).Error(err)
			return nil, err
		}
	case "default":
		rg.Type = RouteTypeDefault
	default:
		err = errors.New("unknown route type: " + rt.Type)
		log.WithField("routes.route.type", rt.Type).Error(err)
		return nil, err
	}
	return rg, nil
}

func (rg *routeGo) MatchDomain(domain string) bool {
	if rg.Type == RouteTypeDefault {
		return true
	}
	if rg.Type == RouteTypeDomain {
		for _, v := range rg.CondDomain.FindAllString(domain, -1) {
			if v == domain {
				return true
			}
		}
	}
	return false
}

func (rg *routeGo) MatchIP(ip net.IP) bool {
	switch rg.Type {
	case RouteTypeIPv4:
		if ip.To4() == nil {
			return false
		}
		return rg.CondIP.Equal(ip)
	case RouteTypeIPv6:
		if ip.To16() == nil {
			return false
		}
		return rg.CondIP.Equal(ip)
	case RouteTypeIPv4Subnet:
		if ip.To4() == nil {
			return false
		}
		return rg.CondIPSubnet.Contains(ip)
	case RouteTypeIPv6Subnet:
		if ip.To16() == nil {
			return false
		}
		return rg.CondIPSubnet.Contains(ip)
	case RouteTypeGeoIP:
		record, err := rg.GeoIPDB.Country(ip)
		if err != nil {
			log.WithField("IP", ip).Warn(err)
			return false
		}
		for _, v := range rg.CondGeoIP {
			if record.Country.IsoCode == v {
				return true
			}
		}
		return false
	case RouteTypeDefault:
		return true
	default:
		return false
	}
}
