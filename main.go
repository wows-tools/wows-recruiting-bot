package main

import (
	"fmt"
	"github.com/kakwa/wows-recruiting-bot/controller"
	"github.com/kakwa/wows-recruiting-bot/model"
	"go.uber.org/zap"
	"golang.org/x/exp/constraints"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"moul.io/zapgorm2"
	"os"
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
	api := controller.NewController(key, server, sugar.With("component", "wows_api"), db)
	api.FillShipMapping()
	err = api.ScrapAllClans()
	if err != nil {
		fmt.Printf(err.Error())
	}
}
