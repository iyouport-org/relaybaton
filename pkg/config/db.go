package config

import (
	"errors"
	"fmt"

	"github.com/iyouport-org/relaybaton/pkg/model"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"     //mysql
	"gorm.io/driver/postgres"  //postgres
	"gorm.io/driver/sqlite"    //sqlite
	"gorm.io/driver/sqlserver" //mssql
	"gorm.io/gorm"
)

type dbType string

const (
	DBTypeMySQL      dbType = "mysql"
	DBTypePostgreSQL dbType = "postgresql"
	DBTypeSQLite3    dbType = "sqlite3"
	DBTypeSQLServer  dbType = "sqlserver"
)

type DBToml struct {
	Type     string `mapstructure:"type"  toml:"type" validate:"required"`
	Username string `mapstructure:"username"  toml:"username" validate:"required"`
	Password string `mapstructure:"password"  toml:"password" validate:"required"`
	Host     string `mapstructure:"host"  toml:"host" validate:"required"`
	Port     int    `mapstructure:"port"  toml:"port" validate:"required"`
	Database string `mapstructure:"database"  toml:"database" validate:"required"`
}

type dbGo struct {
	Type dbType
	DB   *gorm.DB
}

func (dbt *DBToml) Init() (dbg *dbGo, err error) {
	dbg = &dbGo{}
	conf := &gorm.Config{
		//Logger: &relaybaton_log.DBLogger{Logger: log.New()},
	}
	var connStr string
	switch dbt.Type {
	case "mysql":
		connStr = fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local", dbt.Username, dbt.Password, dbt.Host, dbt.Database)
		dbg.Type = DBTypeMySQL
		dbg.DB, err = gorm.Open(mysql.Open(connStr), conf)
	case "postgresql":
		connStr = fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s", dbt.Host, dbt.Port, dbt.Username, dbt.Database, dbt.Password)
		dbg.Type = DBTypePostgreSQL
		dbg.DB, err = gorm.Open(postgres.Open(connStr), conf)
	case "sqlite3":
		connStr = dbt.Database
		dbg.Type = DBTypeSQLite3
		dbg.DB, err = gorm.Open(sqlite.Open(connStr), conf)
	case "sqlserver":
		connStr = fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s", dbt.Username, dbt.Password, dbt.Host, dbt.Port, dbt.Database)
		dbg.Type = DBTypeSQLServer
		dbg.DB, err = gorm.Open(sqlserver.Open(connStr), conf)
	default:
		err = errors.New("unknown GeoIPDB Type: " + dbt.Type)
		log.WithField("db.type", dbt.Type).Error(err)
		return nil, err
	}
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

	err = dbg.DB.AutoMigrate(&model.User{})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	err = dbg.DB.AutoMigrate(&model.Log{})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	err = dbg.DB.AutoMigrate(&model.Meta{})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	err = dbg.DB.AutoMigrate(&model.Notice{})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	err = dbg.DB.AutoMigrate(&model.Plan{})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return dbg, err
}
