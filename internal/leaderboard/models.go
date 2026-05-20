package leaderboard

import (
	"time"

	"github.com/google/uuid"
)

const (
	DefaultMode   = "global"
	DefaultLimit  = 50
	MaxLimit      = 100
	DefaultOffset = 0
)

type Entry struct {
	Rank        int       `json:"rank"`
	UserID      uuid.UUID `json:"userId"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	AvatarURL   string    `json:"avatarUrl"`
	Country     string    `json:"country"`
	CountryCode string    `json:"countryCode"`
	Rating      int       `json:"rating"`
	RatedGames  int       `json:"ratedGames"`
	Wins        int       `json:"wins"`
	Losses      int       `json:"losses"`
	Draws       int       `json:"draws"`
	WinRate     float64   `json:"winRate"`
}

type Page struct {
	Mode       string  `json:"mode"`
	Limit      int     `json:"limit"`
	Offset     int     `json:"offset"`
	Total      int     `json:"total"`
	Items      []Entry `json:"items"`
	ViewerRank *Entry  `json:"viewerRank,omitempty"`
	Stats      Stats   `json:"stats"`
}

type Stats struct {
	QueuedPlayers             int       `json:"queuedPlayers"`
	ActiveMatchPlayers        int       `json:"activeMatchPlayers"`
	RankedPlayers             int       `json:"rankedPlayers"`
	MatchesToday              int       `json:"matchesToday"`
	AverageDailyMatches       float64   `json:"averageDailyMatches"`
	MatchesTodayGrowthPercent float64   `json:"matchesTodayGrowthPercent"`
	CachedAt                  time.Time `json:"cachedAt"`
}
