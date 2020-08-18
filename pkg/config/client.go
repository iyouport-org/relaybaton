package config

type ClientTOML struct {
	Port            int    `mapstructure:"port" toml:"port" validate:"numeric,gte=0,lte=65535,required,nefield=HTTPPort"`
	HTTPPort        int    `mapstructure:"http_port" toml:"http_port" validate:"numeric,gte=0,lte=65535,required,nefield=TransparentPort"`
	TransparentPort int    `mapstructure:"transparent_port" toml:"transparent_port" validate:"numeric,gte=0,lte=65535,required,nefield=Port"`
	Server          string `mapstructure:"server"  toml:"server" validate:"hostname,required"`
	Username        string `mapstructure:"username" toml:"username" validate:"required"`
	Password        string `mapstructure:"password" toml:"password" validate:"required"`
	ProxyAll        bool   `mapstructure:"proxy_all" toml:"proxy_all"`
}

type ClientGo struct {
	Port            uint16
	HTTPPort        uint16
	TransparentPort uint16
	Server          string
	Username        string
	Password        string
	ProxyAll        bool
}

func (ct *ClientTOML) Init() (cg *ClientGo, err error) {
	return &ClientGo{
		Port:            uint16(ct.Port),
		HTTPPort:        uint16(ct.HTTPPort),
		TransparentPort: uint16(ct.TransparentPort),
		Server:          ct.Server,
		Username:        ct.Username,
		Password:        ct.Password,
		ProxyAll:        ct.ProxyAll,
	}, nil
}
