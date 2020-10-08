package model

import "gorm.io/gorm"

type Meta struct {
	gorm.Model
	Title string
	Desc  string
}
