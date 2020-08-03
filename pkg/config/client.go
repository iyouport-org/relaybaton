package config

type ClientTOML struct {
	Port     int    `mapstructure:"port" toml:"port" validate:"numeric"`
	Server   string `mapstructure:"server"  toml:"server" validate:"hostname"`
	Username string `mapstructure:"username" toml:"username" `
	Password string `mapstructure:"password" toml:"password" `
	ProxyAll bool   `mapstructure:"proxy_all" toml:"proxy_all" `
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
