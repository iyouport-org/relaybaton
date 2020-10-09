package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	RoleUser uint = iota
	RoleAdmin
	RoleNone
)

type User struct {
	gorm.Model
	Username    string `gorm:"unique;not null"`
	Role        uint   `gorm:"not null;default:0"`
	Plan        Plan   `gorm:"foreignKey:PlanID"`
	PlanID      uint
	Password    string    `gorm:"not null"`
	TrafficUsed uint      `gorm:"not null"`
	PlanStart   time.Time `gorm:"not null"`
	PlanReset   time.Time `gorm:"not null"`
	PlanEnd     time.Time `gorm:"not null"`
}
