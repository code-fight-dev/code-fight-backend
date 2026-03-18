package users

import (
	"time"

	"github.com/google/uuid"
)

type AvatarSource string

const (
	AvatarSourceNone     AvatarSource = "none"
	AvatarSourceProvider AvatarSource = "provider"
	AvatarSourceCustom   AvatarSource = "custom"
)

type Profile struct {
	ID                uuid.UUID         `json:"id"`
	Username          string            `json:"username"`
	DisplayName       string            `json:"displayName"`
	Bio               string            `json:"bio"`
	Country           string            `json:"country"`
	CountryCode       string            `json:"countryCode"`
	StateProvince     string            `json:"stateProvince"`
	City              string            `json:"city"`
	AvatarURL         string            `json:"avatarUrl"`
	ProviderAvatarURL string            `json:"providerAvatarUrl"`
	AvatarSource      AvatarSource      `json:"avatarSource"`
	AuthProvider      string            `json:"authProvider"`
	CreatedAt         time.Time         `json:"createdAt"`
	Stats             ProfileStats      `json:"stats"`
	EloHistory        []EloHistoryPoint `json:"eloHistory"`
	TopLanguages      []TopLanguage     `json:"topLanguages"`
	RecentMatches     []RecentMatch     `json:"recentMatches"`
	Achievements      []Achievement     `json:"achievements"`
}

type ProfileStats struct {
	EloRating              int     `json:"eloRating"`
	GlobalRank             *int    `json:"globalRank"`
	GlobalPlayersCount     *int    `json:"globalPlayersCount"`
	RegionalRank           *int    `json:"regionalRank"`
	RegionalPlayersCount   *int    `json:"regionalPlayersCount"`
	WinRate                float64 `json:"winRate"`
	TotalMatches           int     `json:"totalMatches"`
	Wins                   int     `json:"wins"`
	Losses                 int     `json:"losses"`
	Draws                  int     `json:"draws"`
	AvgSolutionTimeSeconds *int    `json:"avgSolutionTimeSeconds"`
	MaxWinStreak           int     `json:"maxWinStreak"`
}

type EloHistoryPoint struct {
	Date          string `json:"date"`
	Delta         int    `json:"delta"`
	Rating        int    `json:"rating"`
	MatchesPlayed int    `json:"matchesPlayed"`
}

type TopLanguage struct {
	Name       string  `json:"name"`
	UsageShare float64 `json:"usageShare"`
}

type RecentMatch struct {
	ID string `json:"id"`
}

type Achievement struct {
	ID string `json:"id"`
}

type UpdateProfileInput struct {
	DisplayName        string `json:"displayName"`
	Bio                string `json:"bio"`
	Country            string `json:"country"`
	CountryCode        string `json:"countryCode"`
	StateProvince      string `json:"stateProvince"`
	City               string `json:"city"`
	AvatarDataURL      string `json:"avatarDataUrl"`
	RemoveCustomAvatar bool   `json:"removeCustomAvatar"`
}
