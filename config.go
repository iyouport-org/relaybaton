package relaybaton

import (
	"github.com/iyouport-org/doh-go"
)

// Config is the struct mapped from the configuration file
type Config struct {
	LogFile string       `mapstructure:"log_file"`
	Client  clientConfig `mapstructure:"client"`
	Server  serverConfig `mapstructure:"server"`
}

type clientConfig struct {
	Server   string `mapstructure:"server"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DoH      string `mapstructure:"doh"`
}

type serverConfig struct {
	Port    int    `mapstructure:"port"`
	Pretend string `mapstructure:"pretend"`
	DoH     string `mapstructure:"doh"`
}

func getDoHProvider(provider string) int {
	if provider == "cloudflare" {
		return doh.CloudflareProvider
	}
	if provider == "quad9" {
		return doh.Quad9Provider
	}
	return -1
}
