package main

import (
	"fmt"
	"github.com/kakwa/wows-recruiting-bot/model"
	"github.com/kakwa/wows-recruiting-bot/wows"
	"golang.org/x/exp/constraints"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	api := wows.NewWowsAPI(key, server)

	db, err := gorm.Open(sqlite.Open("wows-recruiting-bo.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	Schemas := []interface{}{
		&model.Player{},
		&model.Clan{},
	}
	// Migrate the schema
	db.AutoMigrate(Schemas...)
	api.FillShipMapping()
	fmt.Printf("Listing all clans\n")
	clanIDs, err := api.ListAllClansIds()
	if err != nil {
		fmt.Printf(err.Error())
	}
	fmt.Printf("%v\n", clanIDs)
	for {
		clanDetails, err := api.GetClansDetails(clanIDs[0:(min(100, len(clanIDs)))])
		if err != nil {
			fmt.Printf(err.Error())
		}

		fmt.Printf("%d\n", len(clanDetails))
		for _, clan := range clanDetails {
			fmt.Printf("%s\n", clan.Tag)
			db.Clauses(clause.OnConflict{UpdateAll: true}).Create(clan)
			fmt.Printf("%v\n", clan.PlayerIDs)
			players, err := api.GetPlayerDetails(clan.PlayerIDs, false)
			if err != nil {
				fmt.Printf("failed to get Players | %s\n", err.Error())
			}
			for _, player := range players {
				player.ClanID = clan.ID
				db.Clauses(clause.OnConflict{UpdateAll: true}).Create(player)
			}
		}

		if len(clanIDs) <= 100 {
			break
		} else {
			clanIDs = clanIDs[100:]
		}
	}

}
