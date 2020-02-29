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

type dbTOML struct {
	Type     string `mapstructure:"type"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Database string `mapstructure:"database"`
}

type dbGo struct {
	Type dbType
	DB   *gorm.DB
}

func (dbt *dbTOML) Init() (dbg *dbGo, err error) {
	dbg = &dbGo{}
	var connStr string
	switch dbt.Type {
	case "mysql":
		connStr = fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local", dbt.Username, dbt.Password, dbt.Host, dbt.Database)
		dbg.Type = DBTypeMySQL
	case "postgresql":
		connStr = fmt.Sprintf("host=%s port=%d User=%s dbname=%s password=%s", dbt.Host, dbt.Port, dbt.Username, dbt.Database, dbt.Password)
		dbg.Type = DBTypePostgreSQL
	case "sqlite3":
		connStr = dbt.Database
		dbg.Type = DBTypeSQLite3
	case "sqlserver":
		connStr = fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s", dbt.Username, dbt.Password, dbt.Host, dbt.Port, dbt.Database)
		dbg.Type = DBTypeSQLServer
	default:
		err = errors.New("unknown GeoIPDB Type: " + dbt.Type)
		log.WithField("db.type", dbt.Type).Error(err)
		return nil, err
	}
	dbg.DB, err = gorm.Open(dbt.Type, connStr)
	if err != nil {
		log.WithFields(log.Fields{
			"db.type":           dbt.Type,
			"connection string": connStr,
		}).Error(err)
		return nil, err
	}
	if dbg.DB == nil {
		err = errors.New("failed to connect to database")
		log.WithFields(log.Fields{
			"db.type":           dbt.Type,
			"connection string": connStr,
		}).Error(err)
		return nil, err
	}
	return dbg, err
}
