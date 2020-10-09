package config

import (
	"net"

	"github.com/go-playground/validator/v10"
	"github.com/iyouport-org/relaybaton/pkg/dns"
	"github.com/iyouport-org/relaybaton/pkg/log"
	"github.com/iyouport-org/relaybaton/pkg/model"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// ConfigTOML is the struct mapped from the configuration file
type ConfigTOML struct {
	Log    *LogTOML    `mapstructure:"log" toml:"log" validate:"required"`
	DNS    *DNSToml    `mapstructure:"dns" toml:"dns" validate:"required"`
	Client *ClientTOML `mapstructure:"client" toml:"client" validate:"-"`
	Server *ServerTOML `mapstructure:"server" toml:"server" validate:"-"`
	DB     *DBToml     `mapstructure:"db" toml:"db" validate:"-"`
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

func (conf *ConfigGo) SaveClient(filename string) error {
	v := viper.New()
	v.Set("client.port", conf.toml.Client.Port)
	v.Set("client.http_port", conf.toml.Client.HTTPPort)
	v.Set("client.redir_port", conf.toml.Client.RedirPort)
	v.Set("client.server", conf.toml.Client.Server)
	v.Set("client.username", conf.toml.Client.Username)
	v.Set("client.password", conf.toml.Client.Password)
	v.Set("client.proxy_all", conf.toml.Client.ProxyAll)
	v.Set("dns.type", conf.toml.DNS.Type)
	v.Set("dns.server", conf.toml.DNS.Server)
	v.Set("dns.addr", conf.toml.DNS.Addr)
	v.Set("log.file", conf.toml.Log.File)
	v.Set("log.level", conf.toml.Log.Level)
	return v.WriteConfigAs(filename)
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

func (conf *ConfigGo) InitClient() error {
	validate := validator.New()
	err := validate.Struct(conf.toml.Client)
	if err != nil {
		logrus.Error(err)
		return err
	}
	conf.Client, err = conf.toml.Client.Init()
	if err != nil {
		logrus.Error(err)
		return err
	}
	return nil
}

func NewConfClient() (conf *ConfigGo, err error) {
	conf = NewConf()
	err = conf.InitClient()
	return
}

func NewConfServer() (conf *ConfigGo, err error) {
	conf = NewConf()
	validate := validator.New()
	err = validate.Struct(conf.toml.Server)
	if err != nil {
		return nil, err
	}
	conf.Server, err = conf.toml.Server.Init()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	err = validate.Struct(conf.toml.DB)
	if err != nil {
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
	if conf.DB != nil {
		conf.DB.DB.AutoMigrate(&model.Log{})
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
