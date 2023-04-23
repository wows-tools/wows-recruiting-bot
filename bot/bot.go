package bot

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/kakwa/wows-recruiting-bot/common"
	"github.com/kakwa/wows-recruiting-bot/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"math/rand"
	"os"
	"sync"
)

type WowsBot struct {
	BotToken        string
	PlayerExitChan  chan common.PlayerExitNotification
	OSSignal        chan os.Signal
	Logger          *zap.SugaredLogger
	Discord         *discordgo.Session
	DB              *gorm.DB
	CommandHandlers map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate)
}

var (
	integerOptionMinValue = 0.0

	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "wows-recruit-test",
			Description: "Test with a random player",
		},
		{
			Name:        "wows-recruit-set-filter",
			Description: "Set the recruitement filter for this channel",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "min-t10",
					Description: "Minimum number of t10",
					MinValue:    &integerOptionMinValue,
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "max-days-last-battle",
					Description: "Number of days since last battle",
					MinValue:    &integerOptionMinValue,
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "min-battles",
					Description: "Minimum number of battles",
					MinValue:    &integerOptionMinValue,
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "min-winrate",
					Description: "Minimum Win Rate (percent)",
					MinValue:    &integerOptionMinValue,
					MaxValue:    100.0,
					Required:    true,
				},
			},
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

func (bot *WowsBot) SetFilter(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}
	var filter model.Filter
	filter.DiscordChannelID = i.ChannelID
	filter.DiscordGuildID = i.GuildID
	filter.MinNumT10 = int(optionMap["min-t10"].IntValue())
	filter.DaysSinceLastBattle = int(optionMap["max-days-last-battle"].IntValue())
	filter.MinNumBattles = int(optionMap["min-battles"].IntValue())
	filter.MinPlayerWR = float64(optionMap["min-winrate"].IntValue()) / 100

	bot.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(filter)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Set filter",
		},
	})
}

func (bot *WowsBot) GetFilter(s *discordgo.Session, i *discordgo.InteractionCreate) {

}

func (bot *WowsBot) AddMonitoredClan(s *discordgo.Session, i *discordgo.InteractionCreate) {

}

func (bot *WowsBot) DeleteMonitoredClan(s *discordgo.Session, i *discordgo.InteractionCreate) {

}

func (bot *WowsBot) ListMonitoredClans(s *discordgo.Session, i *discordgo.InteractionCreate) {

}

func (bot *WowsBot) ReplaceMonitoredClans(s *discordgo.Session, i *discordgo.InteractionCreate) {

}

func NewWowsBot(botToken string, logger *zap.SugaredLogger, db *gorm.DB, playerExitChan chan common.PlayerExitNotification, botChanOSSig chan os.Signal) *WowsBot {
	var bot WowsBot
	bot.PlayerExitChan = playerExitChan
	bot.Logger = logger
	bot.DB = db
	bot.OSSignal = botChanOSSig

	bot.CommandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"wows-recruit-test":       bot.TestOutput,
		"wows-recruit-set-filter": bot.SetFilter,
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + botToken)
	if err != nil {
		bot.Logger.Errorf("error creating Discord session,", err)
		return nil
	}

	dg.AddHandler(bot.LoggedInBot)

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

func (bot *WowsBot) FilterMatch(filter model.Filter, player model.Player, clan model.Clan) bool {
	// TODO
	return true
}

func (bot *WowsBot) StartBot(wg *sync.WaitGroup) {
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

	wg.Add(1)
	defer wg.Done()
	for {
		select {
		case change := <-bot.PlayerExitChan:
			filters := make([]model.Filter, 0)
			bot.DB.Find(&filters)
			for _, filter := range filters {
				if bot.FilterMatch(filter, change.Player, change.Clan) {
					bot.SendPlayerExitMessage(change.Player, change.Clan, filter.DiscordChannelID)
				}
			}
		case <-bot.OSSignal:
			bot.Logger.Infof("bot received exit signal")
			bot.Logger.Infof("Removing commands...")

			for _, v := range registeredCommands {
				err := s.ApplicationCommandDelete(s.State.User.ID, "", v.ID)
				if err != nil {
					bot.Logger.Errorf("Cannot delete '%v' command: %v", v.Name, err)
				}
			}
			s.Close()
			return
		}
	}
}
