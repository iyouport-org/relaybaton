package config

type ClientTOML struct {
	Port      int    `mapstructure:"port" toml:"port" validate:"numeric,gte=0,lte=65535"`
	HTTPPort  int    `mapstructure:"http_port" toml:"http_port" validate:"numeric,gte=0,lte=65535"`
	RedirPort int    `mapstructure:"redir_port" toml:"redir_port" validate:"numeric,gte=0,lte=65535"`
	Server    string `mapstructure:"server"  toml:"server" validate:"hostname,required"`
	Username  string `mapstructure:"username" toml:"username" validate:"required"`
	Password  string `mapstructure:"password" toml:"password" validate:"required"`
	ProxyAll  bool   `mapstructure:"proxy_all" toml:"proxy_all"`
}

type ClientGo struct {
	Port      uint16
	HTTPPort  uint16
	RedirPort uint16
	Server    string
	Username  string
	Password  string
	ProxyAll  bool
}

func (ct *ClientTOML) Init() (cg *ClientGo, err error) {
	return &ClientGo{
		Port:      uint16(ct.Port),
		HTTPPort:  uint16(ct.HTTPPort),
		RedirPort: uint16(ct.RedirPort),
		Server:    ct.Server,
		Username:  ct.Username,
		Password:  ct.Password,
		ProxyAll:  ct.ProxyAll,
	}, nil
}
