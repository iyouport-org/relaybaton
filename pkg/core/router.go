package core

import (
	"bytes"
	"compress/flate"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/emirpasic/gods/maps/hashmap"
	"github.com/iyouport-org/relaybaton/pkg/config"
	"github.com/mholt/archiver"
	"github.com/oschwald/geoip2-golang"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
)

var reservedIP []*net.IPNet
var IsMobile bool

type Router struct {
	on             bool
	compressedPath string
	mmdbPath       string
	mutex          sync.RWMutex
	GeoIPDB        *geoip2.Reader
	hashMap        *hashmap.Map
	mutexMap       sync.RWMutex
	conf           *config.ConfigGo
}

func NewRouter(conf *config.ConfigGo) (*Router, error) {
	if conf.Client.ProxyAll {
		return &Router{
			on:      false,
			GeoIPDB: nil,
			hashMap: hashmap.New(),
		}, nil
	}
	router := &Router{
		on:      true,
		GeoIPDB: nil,
		conf:    conf,
		hashMap: hashmap.New(),
	}
	var exPath string
	if !IsMobile {
		ex, err := os.Executable()
		if err != nil {
			log.Error(err)
			return nil, err
		}
		exPath = filepath.Dir(ex)
	} else {
		exPath = "/data/data/org.iyouport.relaybaton_mobile/files/"
	}
	router.compressedPath = exPath + "/geoip.mmdb.tar.gz"
	router.mmdbPath = exPath + "/geoip.mmdb"
	log.Debug(exPath) //test
	_, err := os.Stat(router.mmdbPath)
	if err != nil {
		if os.IsNotExist(err) {
			router.SwitchOff()
			return router, nil
		} else {
			log.Error(err)
			return nil, err
		}
	}
	router.GeoIPDB, err = geoip2.Open(router.mmdbPath)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return router, nil
}

func (router *Router) SwitchOn() {
	router.mutex.Lock()
	defer router.mutex.Unlock()
	router.on = true
}

func (router *Router) SwitchOff() {
	router.mutex.Lock()
	defer router.mutex.Unlock()
	router.on = false
}

func (router *Router) Select(ip net.IP) bool {
	result, ok := router.getCache(ip)
	if !ok {
		result = router.getGeoIP(ip)
		router.setCache(ip, result)
	}
	return result
}

func (router *Router) getGeoIP(ip net.IP) bool {
	router.mutex.RLock()
	defer router.mutex.RUnlock()
	if isReservedIP(ip) {
		log.Debug("reserved")
		return false
	}
	if !router.on {
		return true
	} else {
		country, err := router.GeoIPDB.Country(ip)
		if err != nil {
			log.Error(err)
			return true
		}
		return country.Country.IsoCode != "CN"
	}
}

func (router *Router) getCache(ip net.IP) (bool, bool) {
	router.mutexMap.RLock()
	defer router.mutexMap.RUnlock()
	v, ok := router.hashMap.Get(ip.String())
	if !ok {
		return true, ok
	}
	return v.(bool), ok
}

func (router *Router) setCache(ip net.IP, result bool) {
	router.mutexMap.Lock()
	defer router.mutexMap.Unlock()
	router.hashMap.Put(ip.String(), result)
}

func (router *Router) RemoveCache(ip net.IP) {
	router.mutexMap.Lock()
	defer router.mutexMap.Unlock()
	router.hashMap.Remove(ip.String())
}

func (router *Router) Download() error {
	log.Debug("Updating")
	router.SwitchOff()
	client := fasthttp.Client{
		Dial: fasthttpproxy.FasthttpSocksDialer(fmt.Sprintf("localhost:%d", router.conf.Client.Port)),
	}
	resp := make([]byte, 1<<22)
	statusCode, body, err := client.Get(resp, "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-Country&license_key=JvbzLLx7qBZT&suffix=tar.gz")
	if err != nil {
		log.Error(err)
		return err
	}
	if statusCode != fasthttp.StatusOK {
		err = errors.New("wrong status code")
		log.WithField("status code", statusCode).Error(err)
		return err
	}
	out, err := os.Create(router.compressedPath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, bytes.NewReader(body))
	if err != nil {
		log.Error(err)
		return err
	}

	gz := &archiver.TarGz{
		Tar: &archiver.Tar{
			OverwriteExisting:      true,
			MkdirAll:               false,
			ImplicitTopLevelFolder: false,
			ContinueOnError:        false,
		},
		CompressionLevel: flate.DefaultCompression,
	}
	err = gz.Walk(router.compressedPath, func(f archiver.File) error {
		log.Debug(f.Name())
		if filepath.Ext(f.Name()) == ".mmdb" {
			if router.GeoIPDB != nil {
				err := router.GeoIPDB.Close()
				if err != nil {
					log.Error(err)
				}
			}
			mmdb, err := os.Create(router.mmdbPath)
			if err != nil {
				return err
			}
			defer mmdb.Close()
			_, err = io.Copy(mmdb, f)
			if err != nil {
				log.Error(err)
				return err
			}
			return nil
		}
		return nil
	})
	if err != nil {
		log.Error(err)
		return err
	}
	router.GeoIPDB, err = geoip2.Open(router.mmdbPath)
	if err != nil {
		log.Error(err)
		return err
	}
	router.SwitchOn()
	log.Debug("Updated")
	return nil
}

func (router *Router) Update() error {
	log.Debug("Checking update")
	f, err := os.Open(router.compressedPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = router.Download()
			if err != nil {
				log.Error(err)
				return err
			}
			return nil
		} else {
			log.Error(err)
			return err
		}
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Error(err)
		return err
	}
	sum := h.Sum(nil)

	client := fasthttp.Client{
		Dial: fasthttpproxy.FasthttpSocksDialer(fmt.Sprintf("localhost:%d", router.conf.Client.Port)),
	}
	resp := make([]byte, 1<<10)
	statusCode, body, err := client.Get(resp, "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-Country&license_key=JvbzLLx7qBZT&suffix=tar.gz.sha256")
	if err != nil {
		log.Error(err)
		return err
	}
	if statusCode != fasthttp.StatusOK {
		err = errors.New("wrong status code")
		log.WithField("status code", statusCode).Error(err)
		return err
	}
	if strings.HasPrefix(string(body), hex.EncodeToString(sum)) {
		return nil
	} else {
		log.Debug("Update found")
		err = router.Download()
		if err != nil {
			log.Error(err)
			return err
		}
		return nil
	}
}

func isReservedIP(ip net.IP) bool {
	if ip.IsInterfaceLocalMulticast() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() || ip.IsLoopback() || ip.IsMulticast() || ip.IsUnspecified() {
		log.Trace("reserved")
		return true
	}
	for _, block := range reservedIP {
		if block.Contains(ip) {
			log.Debug(block.String())
			return true
		}
	}
	return false
}

func init() {
	for _, cidr := range []string{
		"0.0.0.0/8",
		"10.0.0.0/8",
		"100.64.0.0/10",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"172.16.0.0/12",
		"192.0.0.0/24",
		"192.0.2.0/24",
		"192.88.99.0/24",
		"192.168.0.0/16",
		"198.18.0.0/15",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"224.0.0.0/4",
		"240.0.0.0/4",
		"255.255.255.255/32",

		"::/0",
		"::/128",
		"::1/128",
		//"::ffff:0:0/96",
		"::ffff:0:0:0/96",
		"64:ff9b::/96",
		"100::/64",
		"2001::/32",
		"2001:20::/28",
		"2001:db8::/32",
		"2002::/16",
		"fc00::/7",
		"fe80::/10",
		"ff00::/8",
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			log.WithField("cidr", cidr).Error(err)
			continue
		}
		reservedIP = append(reservedIP, block)
	}
}
