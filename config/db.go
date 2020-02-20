package config

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

type dbType string

const (
	DBTypeMySQL      dbType = "mysql"
	DBTypePostgreSQL dbType = "postgresql"
	DBTypeSQLite3    dbType = "sqlite3"
	DBTypeSQLServer  dbType = "sqlserver"
)

type dbConfig struct {
	TypeString string `mapstructure:"type"`
	Username   string `mapstructure:"username"`
	Password   string `mapstructure:"password"`
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	Database   string `mapstructure:"database"`
	Type       dbType
	DB         *gorm.DB
}

func (dbc dbConfig) Init() error {
	err := dbc.validate()
	if err != nil {
		log.Error(err)
		return err
	}
	dbc.DB, err = dbc.getDB()
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (dbc dbConfig) validate() error {
	return nil
}

func (dbc dbConfig) getDB() (*gorm.DB, error) {
	var connStr string
	switch dbc.TypeString {
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
		log.WithField("dialect", dbc.TypeString).Debug(err)
		return nil, err
	}
	db, err := gorm.Open(dbc.TypeString, connStr)
	if err != nil {
		log.WithFields(log.Fields{
			"dialect":           dbc.TypeString,
			"connection string": connStr,
		}).Error(err)
		return nil, err
	}
	if db == nil {
		err = errors.New("failed to connect to database")
		log.WithFields(log.Fields{
			"dialect":           dbc.TypeString,
			"connection string": connStr,
		}).Error(err)
		return nil, err
	}
	return db, err
}
