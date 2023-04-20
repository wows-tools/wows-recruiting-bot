package main

import (
	"fmt"
	"github.com/kakwa/wows-recruiting-bot/bot"
	"github.com/kakwa/wows-recruiting-bot/common"
	"github.com/kakwa/wows-recruiting-bot/controller"
	"github.com/kakwa/wows-recruiting-bot/model"
	"go.uber.org/zap"
	"golang.org/x/exp/constraints"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"moul.io/zapgorm2"
	"os"
	"os/signal"
	"syscall"
)

func min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func main() {

	key := os.Getenv("WOWS_WOWSAPIKEY")
	server := os.Getenv("WOWS_REALM")
	debug := os.Getenv("WOWS_DEBUG")
	botToken := os.Getenv("WOWS_DISCORD_TOKEN")

	var loggerConfig zap.Config
	if debug == "true" {
		loggerConfig = zap.NewDevelopmentConfig()
	} else {
		loggerConfig = zap.NewProductionConfig()
	}

	logger, err := loggerConfig.Build()
	if err != nil {
		fmt.Printf("Error initializing logger: %s\n", err.Error())
		os.Exit(-1)
	}
	defer logger.Sync()
	glogger := zapgorm2.New(logger)
	sugar := logger.Sugar()

	db, err := gorm.Open(sqlite.Open("wows-recruiting-bot.db"), &gorm.Config{Logger: glogger})
	if err != nil {
		panic("failed to connect database")
	}

	Schemas := []interface{}{
		&model.Player{},
		&model.PreviousClan{},
		&model.Clan{},
		&model.Filter{},
	}

	// Migrate the schema
	db.AutoMigrate(Schemas...)

	ch := make(chan common.PlayerExitNotification)
	api := controller.NewController(key, server, sugar.With("component", "wows_api"), db, ch)
	disbot := bot.NewWowsBot(botToken, sugar.With("component", "discord_bot"), db, ch)
	go disbot.StartBot()

	api.FillShipMapping()
	err = api.ScrapAllClans()
	if err != nil {
		fmt.Printf(err.Error())
	}
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
