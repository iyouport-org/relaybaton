package relaybaton

type Config struct {
	LogFile string       `mapstructure:"log_file"`
	Client  ClientConfig `mapstructure:"client"`
	Server  ServerConfig `mapstructure:"server"`
}

type ClientConfig struct {
	Server   string `mapstructure:"server"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type ServerConfig struct {
	Port    int    `mapstructure:"port"`
	Pretend string `mapstructure:"pretend"`
}
