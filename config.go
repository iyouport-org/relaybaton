package relaybaton

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mssql"    //mssql
	_ "github.com/jinzhu/gorm/dialects/mysql"    //mysql
	_ "github.com/jinzhu/gorm/dialects/postgres" //postgres
	_ "github.com/jinzhu/gorm/dialects/sqlite"   //sqlite
	log "github.com/sirupsen/logrus"
)

// Config is the struct mapped from the configuration file
type Config struct {
	LogFile string       `mapstructure:"log_file"`
	Client  clientConfig `mapstructure:"client"`
	Server  serverConfig `mapstructure:"server"`
	DNS     dnsConfig    `mapstructure:"dns"`
	DB      dbConfig     `mapstructure:"db"`
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

type dnsConfig struct {
	Type         string `mapstructure:"type"`
	Server       string `mapstructure:"server"`
	Addr         string `mapstructure:"addr"`
	LocalResolve bool   `mapstructure:"local_resolve"`
}

type dbConfig struct {
	Type     string `mapstructure:"type"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Database string `mapstructure:"database"`
}

func (dbc dbConfig) getDB() (*gorm.DB, error) {
	var connStr string
	switch dbc.Type {
	case "mysql":
		connStr = fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local", dbc.Username, dbc.Password, dbc.Host, dbc.Database)
	case "postgresql":
		connStr = fmt.Sprintf("host=%s port=%d User=%s dbname=%s password=%s", dbc.Host, dbc.Port, dbc.Username, dbc.Database, dbc.Password)
	case "sqlite3":
		connStr = dbc.Database
	case "sqlserver":
		connStr = fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s", dbc.Username, dbc.Password, dbc.Host, dbc.Port, dbc.Database)
	default:
		err := errors.New("unknown database dialect")
		log.WithField("dialect", dbc.Type).Debug(err)
		return nil, err
	}
	db, err := gorm.Open(dbc.Type, connStr)
	if err != nil {
		log.WithFields(log.Fields{
			"dialect":           dbc.Type,
			"connection string": connStr,
		}).Error(err)
		return nil, err
	}
	if db == nil {
		err = errors.New("failed to connect to database")
		log.WithFields(log.Fields{
			"dialect":           dbc.Type,
			"connection string": connStr,
		}).Error(err)
		return nil, err
	}
	return db, err
}
