package core

import (
	_ "github.com/jinzhu/gorm/dialects/mssql"    //mssql
	_ "github.com/jinzhu/gorm/dialects/mysql"    //mysql
	_ "github.com/jinzhu/gorm/dialects/postgres" //postgres
	_ "github.com/jinzhu/gorm/dialects/sqlite"   //sqlite
	"time"
)

type User struct {
	ID               uint      `gorm:"primary_key"`
	Username         string    `gorm:"unique;not null"`
	Password         string    `gorm:"not null"`
	BandwidthLimit   uint      `gorm:"not null"`
	TrafficLimit     uint      `gorm:"not null"`
	TrafficUsed      uint      `gorm:"not null"`
	TrafficStartTime time.Time `gorm:"not null"`
	TrafficNextReset time.Time `gorm:"not null"`
	TrafficEndTime   time.Time `gorm:"not null"`
}
