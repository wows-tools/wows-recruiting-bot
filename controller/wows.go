package controller

import (
	"context"
	"errors"
	"github.com/IceflowRE/go-wargaming/v3/wargaming"
	"github.com/IceflowRE/go-wargaming/v3/wargaming/wows"
	"github.com/kakwa/wows-recruiting-bot/model"
	"github.com/pemistahl/lingua-go"
	"go.uber.org/zap"
	"golang.org/x/exp/constraints"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

type Controller struct {
	client      *wargaming.Client
	ShipMapping map[int]int
	Realm       wargaming.Realm
	Detector    lingua.LanguageDetector
	Logger      *zap.SugaredLogger
	DB          *gorm.DB
}

func min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func difference(a, b []int) []int {
	mb := make(map[int]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []int
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

func NewController(key string, realm string, logger *zap.SugaredLogger, db *gorm.DB) *Controller {
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
	return &Controller{
		client:      wargaming.NewClient(key, &wargaming.ClientOptions{HTTPClient: &http.Client{Timeout: 10 * time.Second}}),
		ShipMapping: make(map[int]int),
		Detector:    detector,
		Realm:       wReam,
		Logger:      logger,
		DB:          db,
	}
}

func (ctl *Controller) FillShipMapping() error {
	ctl.Logger.Debugf("Start filling ship mapping")
	client := ctl.client
	respSize := 9999
	pageNo := 1
	for respSize != 0 {
		res, _, err := client.Wows.EncyclopediaShips(context.Background(), wargaming.RealmEu, &wows.EncyclopediaShipsOptions{
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
			ctl.ShipMapping[*ship.ShipId] = *ship.Tier
		}
	}
	ctl.Logger.Debugf("Finish filling ship mapping")
	return nil

}

func (ctl *Controller) GetPlayerT10Count(playerId int) (int, error) {
	ctl.Logger.Debugf("Start getting T10 ship count for player %d", playerId)
	realm := ctl.Realm
	client := ctl.client
	ret := 0
	inGarage := "1"
	res, _, err := client.Wows.ShipsStats(context.Background(), realm, playerId, &wows.ShipsStatsOptions{
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
		shipTier, ok := ctl.ShipMapping[*ship.ShipId]
		if !ok {
			continue
		}
		if shipTier == 10 {
			ret++
		}
	}
	ctl.Logger.Debugf("Finish getting T10 ship count for player %d", playerId)
	return ret, nil
}

func (ctl *Controller) GetPlayerDetails(playerIds []int, withT10 bool) ([]*model.Player, error) {
	ctl.Logger.Debugf("Start getting player details for players %v", playerIds)
	realm := ctl.Realm
	client := ctl.client
	var ret []*model.Player
	res, err := client.Wows.AccountInfo(context.Background(), realm, playerIds, &wows.AccountInfoOptions{
		Fields: []string{"account_id", "created_at", "hidden_profile", "last_battle_time", "logout_at", "nickname", "statistics.pvp.wins", "statistics.pvp.battles", "statistics.battles"},
	})
	if err != nil {
		return nil, err
	}

	for _, playerData := range res {
		if playerData == nil {
			continue
		}

		T10Count := 0
		if withT10 {
			T10Count, err = ctl.GetPlayerT10Count(*playerData.AccountId)
			if err != nil {
				T10Count = 0
			}
		}
		var battles int
		var win int
		if playerData.Statistics == nil || playerData.Statistics.Pvp == nil || playerData.Statistics.Pvp.Battles == nil || playerData.Statistics.Pvp.Wins == nil {
			battles = 1
			win = 0
			ctl.Logger.Debugf("no stats for player %s[%d]", *playerData.Nickname, *playerData.AccountId)
		} else {
			battles = *playerData.Statistics.Pvp.Battles
			win = *playerData.Statistics.Pvp.Wins
		}
		player := &model.Player{
			ID:                  *playerData.AccountId,
			Nick:                *playerData.Nickname,
			AccountCreationDate: playerData.CreatedAt.Time,
			LastBattleDate:      playerData.LastBattleTime.Time,
			LastLogoutDate:      playerData.LogoutAt.Time,
			Battles:             battles,
			WinRate:             float64(win) / float64(battles),
			NumberT10:           T10Count,
			HiddenProfile:       *playerData.HiddenProfile,
			Tracked:             false,
		}
		ret = append(ret, player)
	}
	ctl.Logger.Debugf("Finish getting player details for players %v", playerIds)
	return ret, nil
}

func (ctl *Controller) ListClansIds(page int) ([]int, error) {
	ctl.Logger.Debugf("Start listing clans page[%d]", page)
	client := ctl.client
	var ret []int
	limit := 100
	res, err := client.Wows.ClansList(context.Background(), EURealm, &wows.ClansListOptions{
		Limit:  &limit,
		PageNo: &page,
		Fields: []string{"clan_id"},
	})
	if err != nil {
		return nil, err
	}
	for _, clan := range res {
		ret = append(ret, *clan.ClanId)
	}
	ctl.Logger.Debugf("Finish listing clans page[%d]", page)
	return ret, nil
}

func (ctl *Controller) GetClansDetails(clanIDs []int) (ret []*model.Clan, err error) {
	client := ctl.client
	clanInfo, err := client.Wows.ClansInfo(context.Background(), EURealm, clanIDs, &wows.ClansInfoOptions{
		Extra:  []string{"members"},
		Fields: []string{"description", "name", "tag", "clan_id", "created_at", "is_clan_disbanded", "updated_at", "members_ids", "leader_id"},
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
			language, _ = ctl.Detector.DetectLanguageOf(*clan.Description)
		} else {
			// Otherwise, use the clan name (but that's quite inaccurate
			DescDataLanguage = "ko"
			language, _ = ctl.Detector.DetectLanguageOf(*clan.Name)
		}
		var players []*model.Player
		for _, memberId := range clan.MembersIds {
			players = append(players, &model.Player{ID: memberId})
		}
		ret = append(ret, &model.Clan{
			ID:           *clan.ClanId,
			Name:         *clan.Name,
			Tag:          *clan.Tag,
			Language:     language.String(),
			LanguageData: DescDataLanguage,
			//Players:      players,
			PlayerIDs:    clan.MembersIds,
			PlayerID:     *clan.LeaderId,
			CreationDate: clan.CreatedAt.Time,
			UpdatedDate:  clan.UpdatedAt.Time,
			Tracked:      false,
		})

	}
	return ret, nil
}

func (ctl *Controller) ScrapAllClans() (err error) {
	ctl.Logger.Debugf("Start scrapping all clans")
	page := 1
	for {
		clanIDs, err := ctl.ListClansIds(page)
		if err != nil {
			return err
		}

		clanDetails, err := ctl.GetClansDetails(clanIDs[0:(min(100, len(clanIDs)))])
		if err != nil {
			return err
		}

		for _, clan := range clanDetails {
			ctl.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(clan)
			ctl.Logger.Debugf("Start getting player details for clan [%s]", clan.Tag)
			players, err := ctl.GetPlayerDetails(clan.PlayerIDs, false)
			if err != nil {
				ctl.Logger.Infof("Failed to get Players: %s", err.Error())
			}
			for _, player := range players {
				player.ClanID = clan.ID
				ctl.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(player)
			}
			ctl.Logger.Debugf("Finish getting player details for clan [%s]", clan.Tag)
		}

		if len(clanIDs) < 100 {
			break
		}
		page++
	}
	ctl.Logger.Debugf("Finish scrapping all clans")
	return nil
}
