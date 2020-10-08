package config

const DEFAULT_ADMIN_USERNAME = "admin"

type ServerTOML struct {
	Port          int    `mapstructure:"port" toml:"port" validate:"numeric,gte=0,lte=65535,required"`
	AdminPassword string `mapstructure:"admin_password" toml:"pretend" validate:"required"`
}

type serverGo struct {
	Port          uint16
	AdminPassword string
}

func (st *ServerTOML) Init() (sg *serverGo, err error) {
	sg = &serverGo{
		Port:          uint16(st.Port),
		AdminPassword: st.AdminPassword,
	}
	return sg, nil
}
