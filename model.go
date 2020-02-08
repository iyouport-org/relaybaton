package relaybaton

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mssql"    //mssql
	_ "github.com/jinzhu/gorm/dialects/mysql"    //mysql
	_ "github.com/jinzhu/gorm/dialects/postgres" //postgres
	_ "github.com/jinzhu/gorm/dialects/sqlite"   //sqlite
)

type User struct {
	gorm.Model
	Username string `gorm:"unique;not null"`
	Password string `gorm:"not null"`
}

type NonceRecord struct {
	gorm.Model
	Username string `gorm:"not null"`
	Nonce    []byte `gorm:"not null"`
}
