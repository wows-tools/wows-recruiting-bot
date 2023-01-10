package model

import (
	"gorm.io/gorm"
	"time"
)

type Player struct {
	gorm.Model
	ID                  int       `gorm:"primaryKey"`
	Nick                string    `gorm:"index"`
	AccountCreationDate time.Time `gorm:"index"`
	LastBattleDate      time.Time `gorm:"index"`
	ClanID              int       `gorm:"index"`
	NumberT10           int       `gorm:"index"`
	Battles             int       `gorm:"index"`
	WinRate             float64   `gorm:"index"`
}
