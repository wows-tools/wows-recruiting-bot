package common

import (
	"github.com/kakwa/wows-recruiting-bot/model"
)

type PlayerExitNotification struct {
	Player model.Player
	Clan   model.Clan
}
