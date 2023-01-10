package main

import (
	"github.com/kakwa/clan_monitoring/wows"
	"os"
)

func main() {

	key := os.Getenv("WOWS_WOWSAPIKEY")

	api := wows.NewWowsAPI(key, "eu")
	api.FillShipMapping()
}
