package matches

import (
	"context"
	"encoding/json"
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

type TaskMode string

const (
	TaskModeNormal TaskMode = "normal"
	TaskModeHard   TaskMode = "hard"
)

type TaskDifficulty string

const (
	TaskDifficultyEasy   TaskDifficulty = "easy"
	TaskDifficultyMedium TaskDifficulty = "medium"
	TaskDifficultyHard   TaskDifficulty = "hard"
)

type JudgeStatus string

const (
	JudgeStatusNotStarted JudgeStatus = "not_started"
	JudgeStatusReady      JudgeStatus = "ready"
	JudgeStatusRunning    JudgeStatus = "running"
	JudgeStatusFinished   JudgeStatus = "finished"
	JudgeStatusFailed     JudgeStatus = "failed"
)

type MatchResultType string

const (
	MatchResultTypePlayer1Win MatchResultType = "player1_win"
	MatchResultTypePlayer2Win MatchResultType = "player2_win"
	MatchResultTypeDraw       MatchResultType = "draw"
	MatchResultTypeCancelled  MatchResultType = "cancelled"
)

type WinningReason string

const (
	WinningReasonAcceptedFaster    WinningReason = "accepted_faster"
	WinningReasonAcceptedMoreTests WinningReason = "accepted_more_tests"
	WinningReasonOpponentFailed    WinningReason = "opponent_failed"
	WinningReasonSurrender         WinningReason = "surrender"
	WinningReasonDraw              WinningReason = "draw"
	WinningReasonCancelled         WinningReason = "cancelled"
)

type Match struct {
	ID                     uuid.UUID        `json:"id"`
	RatingMode             string           `json:"rating_mode"`
	TaskMode               TaskMode         `json:"task_mode"`
	TaskID                 *uuid.UUID       `json:"task_id,omitempty"`
	TaskDifficultySnapshot *TaskDifficulty  `json:"task_difficulty_snapshot,omitempty"`
	Player1ID              uuid.UUID        `json:"player1_id"`
	Player2ID              uuid.UUID        `json:"player2_id"`
	Player1Ready           bool             `json:"player1_ready"`
	Player2Ready           bool             `json:"player2_ready"`
	WinnerID               *uuid.UUID       `json:"winner_id,omitempty"`
	Status                 MatchStatus      `json:"status"`
	JudgeStatus            JudgeStatus      `json:"judge_status"`
	ResultType             *MatchResultType `json:"result_type,omitempty"`
	WinningReason          *WinningReason   `json:"winning_reason,omitempty"`
	IsRated                bool             `json:"is_rated"`
	RatingApplied          bool             `json:"rating_applied"`
	Player1RatingDelta     *int             `json:"player1_rating_delta,omitempty"`
	Player2RatingDelta     *int             `json:"player2_rating_delta,omitempty"`

	Player1Score    int        `json:"player1_score"`
	Player2Score    int        `json:"player2_score"`
	Player1Attempts int        `json:"player1_attempts"`
	Player2Attempts int        `json:"player2_attempts"`
	Player1Solved   bool       `json:"player1_solved"`
	Player2Solved   bool       `json:"player2_solved"`
	Player1SolvedAt *time.Time `json:"player1_solved_at,omitempty"`
	Player2SolvedAt *time.Time `json:"player2_solved_at,omitempty"`

	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type CreateMatchInput struct {
	Player1ID  uuid.UUID `json:"player1_id"`
	Player2ID  uuid.UUID `json:"player2_id"`
	RatingMode string    `json:"rating_mode"`
	TaskMode   TaskMode  `json:"task_mode"`
	IsRated    bool      `json:"is_rated"`
}

type JoinQueueInput struct {
	UserID     uuid.UUID `json:"user_id"`
	RatingMode string    `json:"rating_mode"`
	TaskMode   TaskMode  `json:"task_mode"`
	IsRated    bool      `json:"is_rated"`
}

type QueuedPlayer struct {
	UserID     uuid.UUID `json:"user_id"`
	RatingMode string    `json:"rating_mode"`
	TaskMode   TaskMode  `json:"task_mode"`
	IsRated    bool      `json:"is_rated"`
	CreatedAt  time.Time `json:"created_at"`
}

type QueueResult struct {
	Status string `json:"status"`
	Match  *Match `json:"match,omitempty"`
}

type PublicStats struct {
	QueuedPlayers      int       `json:"queuedPlayers"`
	ActiveMatchPlayers int       `json:"activeMatchPlayers"`
	CachedAt           time.Time `json:"cachedAt"`
}

type StartMatchInput struct {
	MatchID  uuid.UUID     `json:"match_id"`
	ActorID  uuid.UUID     `json:"actor_id"`
	Duration time.Duration `json:"duration"`
}

type FinishMatchInput struct {
	MatchID       uuid.UUID       `json:"match_id"`
	WinnerID      *uuid.UUID      `json:"winner_id,omitempty"`
	ResultType    MatchResultType `json:"result_type"`
	WinningReason WinningReason   `json:"winning_reason"`
	FinishedAt    *time.Time      `json:"finished_at,omitempty"`
	JudgeStatus   *JudgeStatus    `json:"judge_status,omitempty"`
}

type CancelMatchInput struct {
	MatchID uuid.UUID `json:"match_id"`
	ActorID uuid.UUID `json:"actor_id"`
}

type SurrenderMatchInput struct {
	MatchID uuid.UUID `json:"match_id"`
	ActorID uuid.UUID `json:"actor_id"`
}

type ListUserMatchesInput struct {
	UserID uuid.UUID `json:"user_id"`
	Limit  int       `json:"limit"`
	Offset int       `json:"offset"`
}

type UserRating struct {
	UserID     uuid.UUID `json:"user_id"`
	Mode       string    `json:"mode"`
	Rating     int       `json:"rating"`
	RatedGames int       `json:"rated_games"`
	Wins       int       `json:"wins"`
	Losses     int       `json:"losses"`
	Draws      int       `json:"draws"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type RatingHistoryEntry struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	MatchID   uuid.UUID `json:"match_id"`
	Mode      string    `json:"mode"`
	OldRating int       `json:"old_rating"`
	NewRating int       `json:"new_rating"`
	Delta     int       `json:"delta"`
	CreatedAt time.Time `json:"created_at"`
}

type TaskSelector interface {
	SelectForMatch(ctx context.Context, mode TaskMode) (SelectedTask, error)
}

type SelectedTask struct {
	TaskID     uuid.UUID
	Difficulty TaskDifficulty
}

type ApplySubmissionProgressInput struct {
	MatchID     uuid.UUID
	UserID      uuid.UUID
	Score       int
	IsAccepted  bool
	SolvedAt    *time.Time
	JudgeStatus *string
}

type MatchSubmission struct {
	ID              uuid.UUID  `json:"id"`
	MatchID         uuid.UUID  `json:"matchId"`
	TaskID          uuid.UUID  `json:"taskId"`
	UserID          uuid.UUID  `json:"userId"`
	Language        string     `json:"language"`
	LanguageVersion string     `json:"languageVersion"`
	Status          string     `json:"status"`
	Verdict         *string    `json:"verdict,omitempty"`
	PassedTests     int        `json:"passedTests"`
	TotalTests      int        `json:"totalTests"`
	Score           int        `json:"score"`
	CreatedAt       time.Time  `json:"createdAt"`
	StartedAt       *time.Time `json:"startedAt,omitempty"`
	FinishedAt      *time.Time `json:"finishedAt,omitempty"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type MatchReplayPermissions struct {
	CanViewReplay     bool `json:"can_view_replay"`
	CanViewSourceCode bool `json:"can_view_source_code"`
}

type MatchReplayPlayer struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url"`
}

type MatchReplayEvent struct {
	UserID  uuid.UUID       `json:"user_id"`
	Seq     int64           `json:"seq"`
	TMs     int64           `json:"t_ms"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type MatchReplaySnapshot struct {
	UserID     uuid.UUID `json:"user_id"`
	Seq        int64     `json:"seq"`
	TMs        int64     `json:"t_ms"`
	Language   string    `json:"language"`
	SourceCode string    `json:"source_code"`
}

type MatchReplaySubmission struct {
	ID                uuid.UUID  `json:"id"`
	MatchID           uuid.UUID  `json:"match_id"`
	TaskID            uuid.UUID  `json:"task_id"`
	UserID            uuid.UUID  `json:"user_id"`
	Language          string     `json:"language"`
	LanguageVersion   string     `json:"language_version"`
	SourceCode        string     `json:"source_code"`
	Status            string     `json:"status"`
	Verdict           *string    `json:"verdict,omitempty"`
	JudgeSubmissionID *string    `json:"judge_submission_id,omitempty"`
	JudgeStatus       *string    `json:"judge_status,omitempty"`
	ErrorMessage      *string    `json:"error_message,omitempty"`
	PassedTests       int        `json:"passed_tests"`
	TotalTests        int        `json:"total_tests"`
	CompileTimeMs     *int       `json:"compile_time_ms,omitempty"`
	RunTimeMs         *int       `json:"run_time_ms,omitempty"`
	PeakMemoryKb      *int       `json:"peak_memory_kb,omitempty"`
	Score             int        `json:"score"`
	ExitCode          *int       `json:"exit_code,omitempty"`
	CompileLog        *string    `json:"compile_log,omitempty"`
	StdoutTruncated   *string    `json:"stdout_truncated,omitempty"`
	StderrTruncated   *string    `json:"stderr_truncated,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type MatchReplayTimeline struct {
	Version     int                     `json:"version"`
	DurationMs  int64                   `json:"duration_ms"`
	Events      []MatchReplayEvent      `json:"events"`
	Snapshots   []MatchReplaySnapshot   `json:"snapshots"`
	Submissions []MatchReplaySubmission `json:"submissions"`
}

type MatchReplay struct {
	Permissions MatchReplayPermissions `json:"permissions"`
	Match       Match                  `json:"match"`
	Players     []MatchReplayPlayer    `json:"players"`
	Timeline    MatchReplayTimeline    `json:"timeline"`
}
