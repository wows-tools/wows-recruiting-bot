package bot

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/kakwa/wows-recruiting-bot/common"
	"github.com/kakwa/wows-recruiting-bot/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
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
		{
			Name:        "wows-recruit-get-filter",
			Description: "Get the current filter for this channel",
		},
		{
			Name:        "wows-recruit-list-clans",
			Description: "Get the list of monitored clans (in a CSV file)",
		},
		{
			Name:        "wows-recruit-replace-clans",
			Description: "Replace all the monitored clans with a list from a csv file",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionAttachment,
					Name:        "clans-csv",
					Description: "CSV file containing the clan tags (clan tags must be the first column)",
					Required:    true,
				},
			},
		},
		{
			Name:        "wows-recruit-add-clan",
			Description: "Add a clan to the list of monitored clans",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "clan-tag",
					Description: "Clan Tag to add",
					Required:    true,
				},
			},
		},
		{
			Name:        "wows-recruit-remove-clan",
			Description: "Remove a clan from the list of monitored clans",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "clan-tag",
					Description: "Clan Tag to remove",
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

func FilterToString(filter model.Filter) string {
	msg := fmt.Sprintf("Minimum Win Rate: %d%% | Minimum number of battles: %d | Minimum number of T10s: %d | Maximum number of days since last battle: %d",
		int(filter.MinPlayerWR*100),
		filter.MinNumBattles,
		filter.MinNumT10,
		filter.DaysSinceLastBattle,
	)
	return msg
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
			Content: "Set filter to: " + FilterToString(filter),
		},
	})
}

func (bot *WowsBot) GetFilter(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var filter model.Filter
	filter.DiscordChannelID = i.ChannelID
	err := bot.DB.First(&filter).Error
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Filter doesn't seem to be set for this channel, please use '/wows-recruit-set-filter'",
			},
		})
		return
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Current filter is: " + FilterToString(filter),
		},
	})
}

func (bot *WowsBot) AddMonitoredClan(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}
	clanTag := optionMap["clan-tag"].StringValue()
	var filter model.Filter
	filter.DiscordChannelID = i.ChannelID
	err := bot.DB.First(&filter).Error
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Filter doesn't seem to be set for this channel, please use '/wows-recruit-set-filter' first",
			},
		})
		return
	}

	var clan model.Clan
	clan.Tag = clanTag
	err = bot.DB.Where("tag = ?", clanTag).First(&clan).Error
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Clan [" + clanTag + "] doesn't seem to exist",
			},
		})
		return
	}

	clan.Tracked = true
	bot.DB.Save(&clan)
	bot.DB.Model(&filter).Association("TrackedClans").Append(&clan)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Clan [" + clanTag + "] added",
		},
	})

}

func (bot *WowsBot) RemoveMonitoredClan(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}
	clanTag := optionMap["clan-tag"].StringValue()
	var filter model.Filter
	filter.DiscordChannelID = i.ChannelID
	err := bot.DB.First(&filter).Error
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Filter doesn't seem to be set for this channel, please use '/wows-recruit-set-filter' first",
			},
		})
		return
	}
	var clan model.Clan
	err = bot.DB.Where("tag = ?", clanTag).First(&clan).Error
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Clan [" + clanTag + "] doesn't seem to exist",
			},
		})
		return
	}

	bot.DB.Model(&filter).Association("TrackedClans").Delete(&clan)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Clan [" + clanTag + "] removed",
		},
	})
}

func (bot *WowsBot) ListMonitoredClans(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var filter model.Filter
	filter.DiscordChannelID = i.ChannelID
	err := bot.DB.Preload("TrackedClans").First(&filter).Error
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Filter doesn't seem to be set for this channel, please use '/wows-recruit-set-filter' first",
			},
		})
		return
	}
	var buf bytes.Buffer
	csvWriter := csv.NewWriter(&buf)
	for _, clan := range filter.TrackedClans {
		csvWriter.Write([]string{
			clan.Tag,
			clan.Name,
			clan.Language,
			clan.CreationDate.String(),
			strconv.Itoa(clan.ID),
		})
	}
	csvWriter.Flush()
	reader := bytes.NewReader(buf.Bytes())
	file := discordgo.File{
		Name:        "monitored_clan_list.csv",
		ContentType: "text/csv",
		Reader:      reader,
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "List of monitored clan in attached file",
			Files:   []*discordgo.File{&file},
		},
	})
}

func (bot *WowsBot) ReplaceMonitoredClans(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var filter model.Filter
	filter.DiscordChannelID = i.ChannelID
	err := bot.DB.Preload("TrackedClans").First(&filter).Error
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Filter doesn't seem to be set for this channel, please use '/wows-recruit-set-filter' first",
			},
		})
		return
	}

	attachements := i.ApplicationCommandData().Resolved.Attachments
	url := ""
	for _, value := range attachements {
		url = value.URL
	}
	resp, err := http.Get(url)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to recover csv file",
			},
		})
		bot.Logger.Errorf("error downloading csv file", err)
		return
	}
	//reader := bytes.NewReader(resp.Body)
	csvReader := csv.NewReader(resp.Body)
	records, err := csvReader.ReadAll()
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to recover csv file",
			},
		})

		bot.Logger.Errorf("Unable to parse file as CSV", err)
	}
	defer resp.Body.Close()

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Replacing clans",
		},
	})

	// Remove all tracked clans
	bot.DB.Model(&filter).Association("TrackedClans").Delete(filter.TrackedClans)

	// Add all clans in the CSV file
	for _, line := range records {
		if len(line) < 1 {
			continue
		}
		var clan model.Clan
		clanTag := line[0]
		err = bot.DB.Where("tag = ?", clanTag).First(&clan).Error
		if err != nil {
			bot.Discord.ChannelMessageSend(i.ChannelID, "Clan ["+clanTag+"] doesn't seem to exist")
			continue
		}

		clan.Tracked = true
		bot.DB.Save(&clan)
		bot.DB.Model(&filter).Association("TrackedClans").Append(&clan)

	}
}

func NewWowsBot(botToken string, logger *zap.SugaredLogger, db *gorm.DB, playerExitChan chan common.PlayerExitNotification, botChanOSSig chan os.Signal) *WowsBot {
	var bot WowsBot
	bot.PlayerExitChan = playerExitChan
	bot.Logger = logger
	bot.DB = db
	bot.OSSignal = botChanOSSig

	bot.CommandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"wows-recruit-test":          bot.TestOutput,
		"wows-recruit-set-filter":    bot.SetFilter,
		"wows-recruit-get-filter":    bot.GetFilter,
		"wows-recruit-add-clan":      bot.AddMonitoredClan,
		"wows-recruit-remove-clan":   bot.RemoveMonitoredClan,
		"wows-recruit-list-clans":    bot.ListMonitoredClans,
		"wows-recruit-replace-clans": bot.ReplaceMonitoredClans,
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
	msg := fmt.Sprintf("%s left clan [%s] | WR: %f%% | Battles: %d | T10s: %d | Last Battle: %s | Stats: https://wows-numbers.com/player/%d,%s/",
		common.Escape(player.Nick),
		common.Escape(clan.Tag),
		player.WinRate * 100.0,
		player.Battles,
		player.NumberT10,
		player.LastBattleDate.String(),
		player.ID,
		player.Nick,
	)

	bot.Logger.Infof("Sending discord message <%s> on channel '%s'", msg, discordChannelID)
	bot.Discord.ChannelMessageSend(discordChannelID, msg)
}

func (bot *WowsBot) FilterMatch(filter model.Filter, player model.Player, clan model.Clan) bool {
	if player.WinRate < filter.MinPlayerWR {
		bot.Logger.Debugf("Player '%s' did not match WR for filter '%s'", player.Nick, filter.DiscordChannelID)
		return false
	}
	now := time.Now()
	minLastBattle := now.Add(time.Duration(-24*filter.DaysSinceLastBattle) * time.Hour)
	if player.LastBattleDate.Before(minLastBattle) {
		bot.Logger.Debugf("Player '%s' did not match last battle date for filter '%s'", player.Nick, filter.DiscordChannelID)
		return false
	}
	if player.NumberT10 < filter.MinNumT10 {
		bot.Logger.Debugf("Player '%s' did not match min T10s for filter '%s'", player.Nick, filter.DiscordChannelID)
		return false
	}
	if player.Battles < filter.MinNumBattles {
		bot.Logger.Debugf("Player '%s' did not match min Battles for filter '%s'", player.Nick, filter.DiscordChannelID)
		return false
	}
	for _, trackedClan := range filter.TrackedClans {
		if trackedClan.ID == clan.ID {
			return true
		}
	}

	bot.Logger.Debugf("Player '%s' did not leave a clan tracked by filter '%s'", player.Nick, filter.DiscordChannelID)
	return false
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
	bot.Logger.Infof("Starting main bot loop")
	for {
		select {
		case change := <-bot.PlayerExitChan:
			filters := make([]model.Filter, 0)
			bot.DB.Preload("TrackedClans").Find(&filters)
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
