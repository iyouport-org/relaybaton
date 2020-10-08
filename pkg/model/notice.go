package model

import "gorm.io/gorm"

type Notice struct {
	gorm.Model
	Title string `gorm:"unique;not null"`
	Text  string
}
