package bot

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/kakwa/wows-recruiting-bot/common"
	"github.com/kakwa/wows-recruiting-bot/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
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

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + botToken)
	if err != nil {
		bot.Logger.Errorf("error creating Discord session,", err)
		return nil
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(bot.messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		bot.Logger.Errorf("error opening connection,", err)
		return nil
	}
	bot.Discord = dg

	return &bot
}

func (bot *WowsBot) StartBot() {
	for {
		select {
		case change := <-bot.PlayerExitChan:
			filters := make([]model.Filter, 0)
			bot.DB.Find(&filters)
			for _, filter := range filters {
				if change.Clan.Language == "French" {
					msg := fmt.Sprintf("%s left clan [%s] | WR: %f%% | Battles %d | Last Battle: %s | Stats: https://wows-numbers.com/player/%d,%s/",
						change.Player.Nick,
						change.Clan.Tag,
						change.Player.WinRate,
						change.Player.Battles,
						change.Player.LastBattleDate.String(),
						change.Player.ID,
						change.Player.Nick,
					)

					bot.Logger.Infof("Sending discord message <%s> on channel '%s'", msg, change.Player.Nick, change.Clan.Tag, filter.DiscordChannelID)
					bot.Discord.ChannelMessageSend(filter.DiscordChannelID, msg)
				}
			}
		}
	}
}

func (bot *WowsBot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}
	// If the message is "ping" reply with "Pong!"
	if m.Content == "ping" {
		_, err := s.ChannelMessageSend(m.ChannelID, "Pong!")
		if err != nil {
			bot.Logger.Errorf("error sending message,", err)
		}
	}
	bot.Logger.Infof("Channel ID %s", m.ChannelID)

	// If the message is "pong" reply with "Ping!"
	if m.Content == "pong" {
		_, err := s.ChannelMessageSend(m.ChannelID, "Ping!")
		if err != nil {
			bot.Logger.Errorf("error sending message,", err)
		}
	}
}