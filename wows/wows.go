package wows

import (
	"context"
	"errors"
	"github.com/IceflowRE/go-wargaming/v3/wargaming"
	"github.com/IceflowRE/go-wargaming/v3/wargaming/wows"
	"github.com/kakwa/clan_monitoring/model"
	"github.com/pemistahl/lingua-go"
	"net/http"
	"time"
)

var (
	EURealm   = wargaming.RealmEu
	NARealm   = wargaming.RealmNa
	AsiaRealm = wargaming.RealmAsia
)

var (
	ErrShipReturnInvalid = errors.New("Invalid return size for ship listing")
	ErrUnknownRealm      = errors.New("Unknown Wows realm/server")
)

func WowsRealm(realmStr string) (wargaming.Realm, error) {
	switch realmStr {
	case "eu":
		return EURealm, nil
	case "na":
		return NARealm, nil
	case "asia":
		return AsiaRealm, nil
	default:
		return nil, ErrUnknownRealm
	}
}

type WowsAPI struct {
	client      *wargaming.Client
	ShipMapping map[int]int
	Realm       wargaming.Realm
	Detector    lingua.LanguageDetector
}

func NewWowsAPI(key string, realm string) *WowsAPI {
	languages := []lingua.Language{
		lingua.English,
		lingua.German,
		lingua.French,
		lingua.Polish,
		lingua.Italian,
		lingua.Spanish,
		lingua.Dutch,
		lingua.Welsh,
		lingua.Danish,
		lingua.Bokmal,
		lingua.Nynorsk,
		lingua.Swedish,
		lingua.Czech,
		lingua.Turkish,
		lingua.Finnish,
		lingua.Hungarian,
		lingua.Catalan,
		lingua.Slovak,
		lingua.Romanian,
		lingua.Portuguese,
		lingua.Russian,
		lingua.Estonian,
		lingua.Bosnian,
		lingua.Lithuanian,
		lingua.Albanian,
		lingua.Macedonian,
		lingua.Icelandic,
		lingua.Ukrainian,
		lingua.Croatian,
		lingua.Slovene,
		lingua.Irish,
		lingua.Chinese,
		lingua.Japanese,
		lingua.Azerbaijani,
		lingua.Latvian,
		lingua.Serbian,
		lingua.Bulgarian,
		lingua.Belarusian,
		lingua.Macedonian,
		lingua.Greek,
		lingua.Hebrew,
		lingua.Vietnamese,
		lingua.Kazakh,
	}

	detector := lingua.NewLanguageDetectorBuilder().
		FromLanguages(languages...).
		Build()
	wReam, err := WowsRealm(realm)
	if err != nil {
		return nil
	}
	return &WowsAPI{
		client:      wargaming.NewClient(key, &wargaming.ClientOptions{HTTPClient: &http.Client{Timeout: 10 * time.Second}}),
		ShipMapping: make(map[int]int),
		Detector:    detector,
		Realm:       wReam,
	}
}

func (wowsAPI *WowsAPI) FillShipMapping() error {
	client := wowsAPI.client
	respSize := 9999
	pageNo := 1
	for respSize != 0 {
		res, err := client.Wows.EncyclopediaShips(context.Background(), wargaming.RealmEu, &wows.EncyclopediaShipsOptions{
			Fields: []string{"ship_id", "tier"},
			PageNo: &pageNo,
		})
		if err != nil && pageNo == 1 {
			return err
		}
		if err != nil {
			// FIXME the go-wargaming library doesn't provide the "meta" part of the response
			// (containing the number of pages and number of ships)
			// so for now, we stop on the first error which is not ideal...
			return nil
		}
		respSize = len(res)
		pageNo++
		for _, ship := range res {
			wowsAPI.ShipMapping[*ship.ShipId] = *ship.Tier
		}
	}
	return nil

}

func (wowsAPI *WowsAPI) GetPlayerT10Count(realm wargaming.Realm, playerId int) (int, error) {
	client := wowsAPI.client
	var ret int
	inGarage := "1"
	res, err := client.Wows.ShipsStats(context.Background(), realm, playerId, &wows.ShipsStatsOptions{
		Fields:   []string{"ship_id"},
		InGarage: &inGarage,
	})
	if err != nil {
		return 0, err
	}

	if len(res) != 1 {
		return 0, ErrShipReturnInvalid
	}
	shipList, ok := res[playerId]

	if !ok {
		return 0, ErrShipReturnInvalid
	}

	for _, ship := range shipList {
		shipTier, ok := wowsAPI.ShipMapping[*ship.ShipId]
		if !ok {
			continue
		}
		if shipTier == 10 {
			ret++
		}
	}
	return ret, nil
}

func (wowsAPI *WowsAPI) ListAllClansIds() ([]int, error) {
	client := wowsAPI.client
	var ret []int
	limit := 100
	page := 1
	for {
		res, err := client.Wows.ClansList(context.Background(), EURealm, &wows.ClansListOptions{
			Limit:  &limit,
			PageNo: &page,
		})
		if err != nil {
			continue
		}
		for _, clan := range res {
			ret = append(ret, *clan.ClanId)
		}
		page++
		if len(res) == 0 {
			break
		}
	}
	return ret, nil
}

func (wowsAPI *WowsAPI) GetClansDetails(clanIDs []int) (ret []*model.Clan, err error) {
	client := wowsAPI.client
	clanInfo, err := client.Wows.ClansInfo(context.Background(), EURealm, clanIDs, &wows.ClansInfoOptions{
		Extra:  []string{"members"},
		Fields: []string{"description", "name", "tag", "clan_id", "created_at", "is_clan_disbanded", "updated_at"},
	})
	if err != nil {
		return nil, err
	}

	for _, clan := range clanInfo {
		// Clan is disbanded, ignore
		if clan.IsClanDisbanded != nil && *clan.IsClanDisbanded {
			continue
		}

		var language lingua.Language
		var DescDataLanguage = "ko"
		if clan.Description != nil && len(*clan.Description) > len(*clan.Name) {
			// We have a long enough description, trying to deduce the language from that

			// It's a bit too short to get a good detection
			if len(*clan.Description) < 20 {
				DescDataLanguage = "ko"
			} else {
				DescDataLanguage = "ok"
			}
			language, _ = wowsAPI.Detector.DetectLanguageOf(*clan.Description)
		} else {
			// Otherwise, use the clan name (but that's quite inaccurate
			DescDataLanguage = "ko"
			language, _ = wowsAPI.Detector.DetectLanguageOf(*clan.Name)
			ret = append(ret, &model.Clan{
				ID:           *clan.ClanId,
				Name:         *clan.Name,
				Tag:          *clan.Tag,
				Language:     language.String(),
				LanguageData: DescDataLanguage,
			})
		}
		// TODO
		// Build the lists of clan models

	}
	return ret, nil
}
