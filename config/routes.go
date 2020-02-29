package config

import (
	"errors"
	"github.com/oschwald/geoip2-golang"
	log "github.com/sirupsen/logrus"
)

type routesTOML struct {
	GeoIPFile string      `mapstructure:"geoip_file"`
	Route     []routeTOML `mapstructure:"route"`
}

type routesGo struct {
	GeoIPDB *geoip2.Reader
	Route   []*routeGo
}

func (rt *routesTOML) Init() (rg *routesGo, err error) {
	rg = &routesGo{}
	rg.GeoIPDB, err = geoip2.Open(rt.GeoIPFile)
	if err != nil {
		log.WithField("routes.geoip_file", rt.GeoIPFile).Error(err)
		return nil, err
	}
	rg.Route = []*routeGo{}
	defaultCount := 0
	for _, v := range rt.Route {
		route, err := v.Init(rg.GeoIPDB)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		rg.Route = append(rg.Route, route)
		if route.Type == RouteTypeDefault {
			defaultCount++
		}
	}
	if defaultCount != 1 {
		err = errors.New("no unique default")
		log.WithField("default count", defaultCount).Debug(err)
		return nil, err
	}
	return rg, nil
}
