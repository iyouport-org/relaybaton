package relaybaton

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
}

type serverConfig struct {
	Port    int    `mapstructure:"port"`
	Pretend string `mapstructure:"pretend"`
}
