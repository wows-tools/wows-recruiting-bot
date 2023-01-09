package main

import (
	"github.com/IceflowRE/go-wargaming/v3/wargaming"
	"github.com/IceflowRE/go-wargaming/v3/wargaming/wows"
	"net/http"
	"os"
	"time"
	"context"
	"fmt"
	"log"
)

var (
	EURealm   = wargaming.RealmEu
	NARealm   = wargaming.RealmNa
	AsiaRealm = wargaming.RealmAsia
)


func main() {
	key := os.Getenv("WOWS_WOWSAPIKEY")
	client :=  wargaming.NewClient(key, &wargaming.ClientOptions{HTTPClient: &http.Client{Timeout: 10 * time.Second}})
	limit := 5
	mode := "startswith"
	res, err := client.Wows.AccountList(context.Background(), EURealm, "kakw", &wows.AccountListOptions{
		Fields: []string{"account_id", "nickname"},
		Type:   wargaming.String(mode),
		Limit:  &limit,
	})
        if err != nil {
                log.Fatal(err)
        }
	fmt.Printf(*res[0].Nickname)
}
