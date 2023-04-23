package model

import (
	"gorm.io/gorm"
	"time"
)

type Filter struct {
	gorm.Model
	DiscordChannelID    string   `gorm:"primaryKey"`
	TrackedClans        []Clan   `gorm:"many2many:filter_tracked_clan;"`
	TrackedPlayers      []Player `gorm:"many2many:filter_tracked_player;"`
	MinPlayerWR         float64
	TimeSinceLastBattle time.Time
	MinNumT10           int
	MinNumBattles       int
	DiscordGuildID      string
}
