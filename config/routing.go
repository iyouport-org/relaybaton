package config

import (
	"github.com/oschwald/geoip2-golang"
	"github.com/pariz/gountries"
	log "github.com/sirupsen/logrus"
	"net"
)

type routingType string

const (
	RoutingTypeDefault routingType = "default"
	RoutingTypeGeoIP   routingType = "geoip"
)

type routingTOML struct {
	Type string `mapstructure:"type"`
	File string `mapstructure:"file"`
	Skip string `mapstructure:"skip"`
}

type routingGo struct {
	Type routingType
	DB   *geoip2.Reader
	Skip gountries.Country
}

func (rt *routingTOML) Init() (rg *routingGo, err error) {
	rg = &routingGo{}
	switch rt.Type {
	case "geoip":
		rg.Type = RoutingTypeGeoIP
		rg.DB, err = geoip2.Open(rt.File)
		if err != nil {
			log.WithField("routing.file", rt.File).Error(err)
			return nil, err
		}
		query := gountries.New()
		rg.Skip, err = query.FindCountryByAlpha(rt.Skip)
		if err != nil {
			log.WithField("routing.skip", rt.Skip).Error(err)
			return nil, err
		}
		return rg, nil
	default:
		rg.Type = RoutingTypeDefault
		return rg, nil
	}
}

func (rg *routingGo) Match(ipAddress net.IP) bool {
	switch rg.Type {
	case RoutingTypeGeoIP:
		return rg.geoIPMatch(ipAddress)
	case RoutingTypeDefault:
		return false
	default:
		return false
	}
}

func (rg *routingGo) geoIPMatch(ipAddress net.IP) bool {
	record, err := rg.DB.Country(ipAddress)
	if err != nil {
		log.WithField("IP", ipAddress.String()).Error(err)
	}
	return record.Country.IsoCode == rg.Skip.Alpha2
}
