package model

import (
	"gorm.io/gorm"
	"time"
)

type Clan struct {
	gorm.Model
	ID           int `gorm:"primaryKey"`
	Name         string
	Tag          string `gorm:"index"`
	Language     string `gorm:"index"`
	LanguageData string
	CreationDate time.Time
	UpdatedDate  time.Time
	Players      []*Player
	PlayerIDs    []int `gorm:"-"`
	ClanLeader   *Player
	PlayerID     int  `gorm:"index"`
	Tracked      bool `gorm:"index"`
}
