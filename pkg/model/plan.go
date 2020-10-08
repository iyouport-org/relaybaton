package model

import "gorm.io/gorm"

type Plan struct {
	gorm.Model
	Name           string `gorm:"unique;not null"`
	BandwidthLimit uint   `gorm:"not null"`
	TrafficLimit   uint   `gorm:"not null"`
}
