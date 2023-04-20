package bot

import (
	"github.com/bwmarrin/discordgo"
        "github.com/kakwa/wows-recruiting-bot/common"
        "go.uber.org/zap"
        "gorm.io/gorm"
	"fmt"
)

type WowsBot struct {
	BotToken       string
	PlayerExitChan chan common.PlayerExitNotification
	Logger         *zap.SugaredLogger
	Discord        *discordgo.Session
        DB             *gorm.DB
}

func NewWowsBot(botToken string, logger *zap.SugaredLogger, db *gorm.DB, playerExitChan chan common.PlayerExitNotification) *WowsBot {
	var bot WowsBot
	bot.PlayerExitChan = playerExitChan
	bot.Logger = logger
	bot.DB = db
	return &bot
}

func (bot *WowsBot)StartBot() {
	for {
		select {
		case change := <-bot.PlayerExitChan:
			fmt.Printf(">>>>>> %s %s\n", change.Player.Nick, change.Clan.Tag)
		}
	}
}
