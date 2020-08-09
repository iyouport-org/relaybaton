package config

import (
	"github.com/go-playground/validator/v10"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net"
	"relaybaton/pkg/dns"
	"relaybaton/pkg/log"
)

// ConfigTOML is the struct mapped from the configuration file
type ConfigTOML struct {
	Log    *LogTOML    `mapstructure:"log" toml:"log" validate:"required"`
	DNS    *DNSToml    `mapstructure:"dns" toml:"dns" validate:"required"`
	Client *ClientTOML `mapstructure:"client" toml:"client" validate:"required_without=Server"`
	Server *ServerTOML `mapstructure:"server" toml:"server"  validate:"required_without=Client"`
	DB     *DBToml     `mapstructure:"db" toml:"db" validate:"required_without=Client"`
}

type ConfigGo struct {
	toml   *ConfigTOML
	Log    *LogGo    //client,server
	DNS    *DNSGo    //client,server
	Client *ClientGo //client
	Server *serverGo //server
	DB     *dbGo     //server
}

func (mc *ConfigTOML) Init() (cg *ConfigGo, err error) {
	validate := validator.New()
	err = validate.Struct(mc)
	if err != nil {
		return nil, err
	}
	cg = &ConfigGo{}
	cg.toml = mc
	cg.Log, err = mc.Log.Init()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	cg.DNS, err = mc.DNS.Init()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return cg, nil
}

func (conf *ConfigGo) Save(filename string) error {
	viper.Set("client.port", conf.toml.Client.Port)
	viper.Set("client.server", conf.toml.Client.Server)
	viper.Set("client.username", conf.toml.Client.Username)
	viper.Set("client.password", conf.toml.Client.Password)
	viper.Set("client.proxy_all", conf.toml.Client.ProxyAll)
	viper.Set("dns.type", conf.toml.DNS.Type)
	viper.Set("dns.server", conf.toml.DNS.Server)
	viper.Set("dns.addr", conf.toml.DNS.Addr)
	viper.Set("log.file", conf.toml.Log.File)
	viper.Set("log.level", conf.toml.Log.Level)
	return viper.WriteConfigAs(filename)
}

func NewConf() *ConfigGo {
	v := viper.GetViper()
	if viper.GetString("config") != "" {
		logrus.Debug(viper.GetString("config"))
		v.SetConfigFile(viper.GetString("config"))
	}
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		logrus.Panic(err)
	}
	logrus.Debug(v.ConfigFileUsed())
	var confTOML ConfigTOML
	err := v.Unmarshal(&confTOML)
	if err != nil {
		logrus.Panic(err)
	}
	conf, err := confTOML.Init()
	if err != nil {
		logrus.Panic(err)
	}

	return conf
}

func NewConfClient() (conf *ConfigGo, err error) {
	conf = NewConf()
	conf.Client, err = conf.toml.Client.Init()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return conf, nil
}

func NewConfServer() (conf *ConfigGo, err error) {
	conf = NewConf()
	conf.Server, err = conf.toml.Server.Init()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	conf.DB, err = conf.toml.DB.Init()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return conf, nil
}

func InitLog(conf *ConfigGo) {
	logrus.SetFormatter(log.XMLFormatter{})
	logrus.SetOutput(conf.Log.File)
	logrus.SetLevel(conf.Log.Level)
	if conf.DB == nil {
		db, err := gorm.Open("sqlite3", "log.db")
		if err != nil {
			logrus.Error(err)
			return
		}
		db.AutoMigrate(&log.Record{})
		logrus.AddHook(log.NewSQLiteHook(db))
	} else {
		conf.DB.DB.AutoMigrate(&log.Record{})
		logrus.AddHook(log.NewSQLiteHook(conf.DB.DB))
	}
}

func InitDNS(conf *ConfigGo) {
	switch conf.DNS.Type {
	case DNSTypeDoT:
		factory := dns.NewDoTResolverFactory(net.Dialer{}, conf.DNS.Server, conf.DNS.Addr, false)
		net.DefaultResolver = factory.GetResolver()
	case DNSTypeDoH:
		factory, err := dns.NewDoHResolverFactory(net.Dialer{}, 1083, conf.DNS.Server, conf.DNS.Addr, false)
		if err != nil {
			logrus.Error(err)
			return
		}
		net.DefaultResolver = factory.GetResolver()
	}
}
