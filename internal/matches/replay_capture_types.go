package matches

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	replayEventsMaxBatchSize     = 200
	replayEventPayloadMaxBytes   = 16 * 1024
	replaySnapshotSourceMaxBytes = 256 * 1024
	replayLanguageMaxLength      = 32
	replayCaptureGracePeriod     = 10 * time.Second
	replayCaptureTimelineVersion = 1
)

type MatchReplayCapturedEvent struct {
	MatchID   uuid.UUID
	UserID    uuid.UUID
	Seq       int64
	TMs       int64
	EventType string
	Payload   json.RawMessage
	CreatedAt time.Time
}

type MatchReplayCapturedSnapshot struct {
	MatchID    uuid.UUID
	UserID     uuid.UUID
	Seq        int64
	TMs        int64
	Language   string
	SourceCode string
	CreatedAt  time.Time
}

type MatchReplayEventIngestItem struct {
	Seq       int64
	TMs       int64
	EventType string
	Payload   json.RawMessage
}

type MatchReplayEventsIngestInput struct {
	MatchID uuid.UUID
	UserID  uuid.UUID
	Events  []MatchReplayEventIngestItem
}

type MatchReplayEventsIngestResult struct {
	Ingested     int   `json:"ingested"`
	Deduplicated int   `json:"deduplicated"`
	LastSeq      int64 `json:"last_seq"`
	ExpectedSeq  int64 `json:"expected_seq"`
}

type MatchReplaySnapshotIngestInput struct {
	MatchID    uuid.UUID
	UserID     uuid.UUID
	Seq        int64
	TMs        int64
	Language   string
	SourceCode string
}

type MatchReplaySnapshotIngestResult struct {
	Inserted    bool  `json:"inserted"`
	LastSeq     int64 `json:"last_seq"`
	ExpectedSeq int64 `json:"expected_seq"`
}
