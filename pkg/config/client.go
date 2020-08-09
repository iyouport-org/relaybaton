package config

type ClientTOML struct {
	Port     int    `mapstructure:"port" toml:"port" validate:"numeric,gte=0,lte=65535,required_with=ClientTOML"`
	Server   string `mapstructure:"server"  toml:"server" validate:"hostname,required_with=ClientTOML"`
	Username string `mapstructure:"username" toml:"username" validate:"required_with=ClientTOML"`
	Password string `mapstructure:"password" toml:"password" validate:"required_with=ClientTOML"`
	ProxyAll bool   `mapstructure:"proxy_all" toml:"proxy_all" validate:"required_with=ClientTOML"`
}

type ClientGo struct {
	Port     uint16
	Server   string
	Username string
	Password string
	ProxyAll bool
}

func (ct *ClientTOML) Init() (cg *ClientGo, err error) {
	return &ClientGo{
		Port:     uint16(ct.Port),
		Server:   ct.Server,
		Username: ct.Username,
		Password: ct.Password,
		ProxyAll: ct.ProxyAll,
	}, nil
}
