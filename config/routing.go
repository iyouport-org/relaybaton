package config

import (
	"errors"
	"github.com/oschwald/geoip2-golang"
	"github.com/pariz/gountries"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
)

type routingType string

const (
	RoutingTypeAll     routingType = "all"
	RoutingTypeGFWList routingType = "gfwlist"
	RoutingTypePAC     routingType = "pac"
	RoutingTypeGeoIP   routingType = "geoip"
)

func (rt routingType) validate() error {
	switch rt {
	case RoutingTypeAll:
		return nil
	case RoutingTypeGFWList:
		return nil
	case RoutingTypePAC:
		return nil
	case RoutingTypeGeoIP:
		return nil
	default:
		err := errors.New("unknown routing type")
		log.WithField("routing.type", rt).Error(err)
		return err
	}
}

type routingConfig struct {
	TypeString      string `mapstructure:"type"`
	FileString      string `mapstructure:"file"`
	SkipString      string `mapstructure:"skip"`
	TypeRoutingType routingType
	DB              *geoip2.Reader
	SkipCountry     gountries.Country
}

func (rc *routingConfig) Init() error {
	err := rc.validate()
	if err != nil {
		log.Error(err)
		return err
	}

	rc.TypeRoutingType = routingType(rc.TypeString)
	rc.DB, err = geoip2.Open(rc.FileString)
	if err != nil {
		log.WithField("routing.file", rc.FileString).Error(err)
		return err
	}
	return nil
}

func (rc *routingConfig) validate() error {
	err := routingType(rc.TypeString).validate()
	if err != nil {
		log.WithField("routing.type", rc.TypeString).Error(err)
		return err
	}
	_, err = os.Stat(rc.FileString)
	if err != nil {
		log.WithField("routing.file", rc.FileString).Error(err)
		return err
	}
	query := gountries.New()
	rc.SkipCountry, err = query.FindCountryByAlpha(rc.SkipString)
	if err != nil {
		log.WithField("routing.skip", rc.SkipString).Error(err)
		return err
	}
	return nil
}

func (rc *routingConfig) Match(ipAddress net.IP) bool {
	return rc.GeoIPMatch(ipAddress)
}

func (rc *routingConfig) GeoIPMatch(ipAddress net.IP) bool {
	record, err := rc.DB.Country(ipAddress)
	if err != nil {
		log.WithField("IP", ipAddress.String()).Error(err)
	}
	return record.Country.IsoCode == rc.SkipCountry.Alpha2
}
