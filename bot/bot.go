package bot

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/kakwa/wows-recruiting-bot/common"
	"github.com/kakwa/wows-recruiting-bot/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"math/rand"
)

type WowsBot struct {
	BotToken        string
	PlayerExitChan  chan common.PlayerExitNotification
	Logger          *zap.SugaredLogger
	Discord         *discordgo.Session
	DB              *gorm.DB
	CommandHandlers map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate)
}

var (
	integerOptionMinValue          = 1.0
	dmPermission                   = false
	defaultMemberPermissions int64 = discordgo.PermissionManageServer

	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "wows-recruit-test",
			Description: "Test the output with a random player",
		},
	}
)

func (bot *WowsBot) TestOutput(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Sending fake player clan exit for testing",
		},
	})
	var player model.Player
	wr := 0.30 + rand.Float64()*0.30
	bot.DB.Where("win_rate > ?", wr).Preload("Clan").Order("win_rate").First(&player)
	clan := model.Clan{
		Tag: "TEST",
	}
	bot.SendPlayerExitMessage(player, clan, i.ChannelID)
}

func NewWowsBot(botToken string, logger *zap.SugaredLogger, db *gorm.DB, playerExitChan chan common.PlayerExitNotification) *WowsBot {
	var bot WowsBot
	bot.PlayerExitChan = playerExitChan
	bot.Logger = logger
	bot.DB = db

	bot.CommandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"wows-recruit-test": bot.TestOutput,
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + botToken)
	if err != nil {
		bot.Logger.Errorf("error creating Discord session,", err)
		return nil
	}

	dg.AddHandler(bot.LoggedInBot)

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

func (bot *WowsBot) LoggedInBot(s *discordgo.Session, r *discordgo.Ready) {
	bot.Logger.Infof("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
}

func (bot *WowsBot) SendPlayerExitMessage(player model.Player, clan model.Clan, discordChannelID string) {
	msg := fmt.Sprintf("%s left clan [%s] | WR: %f%% | Battles %d | Last Battle: %s | Stats: https://wows-numbers.com/player/%d,%s/",
		player.Nick,
		clan.Tag,
		player.WinRate,
		player.Battles,
		player.LastBattleDate.String(),
		player.ID,
		player.Nick,
	)

	bot.Logger.Infof("Sending discord message <%s> on channel '%s'", msg, discordChannelID)
	bot.Discord.ChannelMessageSend(discordChannelID, msg)
}

func (bot *WowsBot) StartBot() {
	bot.Logger.Infof("Adding commands...")
	s := bot.Discord

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := bot.CommandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, "", v)
		if err != nil {
			bot.Logger.Errorf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	for {
		select {
		case change := <-bot.PlayerExitChan:
			filters := make([]model.Filter, 0)
			bot.DB.Find(&filters)
			for _, filter := range filters {
				if change.Clan.Language == "French" {
					bot.SendPlayerExitMessage(change.Player, change.Clan, filter.DiscordChannelID)
				}
			}
		}
	}
	bot.Logger.Infof("Removing commands...")
	// // We need to fetch the commands, since deleting requires the command ID.
	// // We are doing this from the returned commands on line 375, because using
	// // this will delete all the commands, which might not be desirable, so we
	// // are deleting only the commands that we added.
	// registeredCommands, err := s.ApplicationCommands(s.State.User.ID, *GuildID)
	// if err != nil {
	// 	log.Fatalf("Could not fetch registered commands: %v", err)
	// }

	for _, v := range registeredCommands {
		err := s.ApplicationCommandDelete(s.State.User.ID, "", v.ID)
		if err != nil {
			bot.Logger.Errorf("Cannot delete '%v' command: %v", v.Name, err)
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
	bot.Logger.Infof("Channel ID '%s', '%s'", m.ChannelID, m.Content)

	// If the message is "pong" reply with "Ping!"
	if m.Content == "pong" {
		_, err := s.ChannelMessageSend(m.ChannelID, "Ping!")
		if err != nil {
			bot.Logger.Errorf("error sending message,", err)
		}
	}
}
