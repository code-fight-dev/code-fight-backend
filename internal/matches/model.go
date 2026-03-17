package matches

import (
	"time"

	"github.com/google/uuid"
)

type MatchStatus string

const (
	MatchStatusPending   MatchStatus = "pending"
	MatchStatusRunning   MatchStatus = "running"
	MatchStatusFinished  MatchStatus = "finished"
	MatchStatusCancelled MatchStatus = "cancelled"
)

type Match struct {
	ID            uuid.UUID   `json:"id"`
	Mode          string      `json:"mode"`
	Player1ID     uuid.UUID   `json:"player1_id"`
	Player2ID     uuid.UUID   `json:"player2_id"`
	WinnerID      *uuid.UUID  `json:"winner_id,omitempty"`
	Status        MatchStatus `json:"status"`
	IsRated       bool        `json:"is_rated"`
	RatingApplied bool        `json:"rating_applied"`
	StartedAt     *time.Time  `json:"started_at,omitempty"`
	FinishedAt    *time.Time  `json:"finished_at,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

type CreateMatchInput struct {
	Player1ID uuid.UUID `json:"player1_id"`
	Player2ID uuid.UUID `json:"player2_id"`
	Mode      string    `json:"mode"`
	IsRated   bool      `json:"is_rated"`
}

type StartMatchInput struct {
	MatchID uuid.UUID `json:"match_id"`
}

type FinishMatchInput struct {
	MatchID  uuid.UUID  `json:"match_id"`
	WinnerID *uuid.UUID `json:"winner_id"`
}

type CancelMatchInput struct {
	MatchID uuid.UUID `json:"match_id"`
}

type ListUserMatchesInput struct {
	UserID uuid.UUID `json:"user_id"`
	Limit  int       `json:"limit"`
	Offset int       `json:"offset"`
}

type UserRating struct {
	UserID      uuid.UUID
	Mode        string
	Rating      int
	RatedGames  int
	Wins        int
	Losses      int
	Draws       int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type RatingHistoryEntry struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	MatchID   uuid.UUID
	Mode      string
	OldRating int
	NewRating int
	Delta     int
	CreatedAt time.Time
}