package config

type clientTOML struct {
	Server   string `mapstructure:"server"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type clientGo struct {
	Server   string
	Port     uint16
	Username string
	Password string
}

func (ct *clientTOML) Init() (cg *clientGo, err error) {
	return &clientGo{
		Server:   ct.Server,
		Port:     uint16(ct.Port),
		Username: ct.Username,
		Password: ct.Password,
	}, nil
}
