package matches

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIngestMatchReplayEventsValidatesInput(t *testing.T) {
	player1 := uuid.New()
	player2 := uuid.New()
	matchID := uuid.New()
	repo := &fakeReplayMatchRepo{fakeMatchServiceRepo: fakeMatchServiceRepo{match: Match{
		ID:        matchID,
		Player1ID: player1,
		Player2ID: player2,
		Status:    MatchStatusRunning,
	}}}
	service := NewService(repo, nil, nil)

	base := MatchReplayEventsIngestInput{
		MatchID: matchID,
		UserID:  player1,
		Events:  []MatchReplayEventIngestItem{{Seq: 1, TMs: 0, EventType: " insert ", Payload: nil}},
	}

	tests := []struct {
		name    string
		mutate  func(*MatchReplayEventsIngestInput)
		wantErr error
	}{
		{name: "empty match", mutate: func(input *MatchReplayEventsIngestInput) { input.MatchID = uuid.Nil }},
		{name: "empty user", mutate: func(input *MatchReplayEventsIngestInput) { input.UserID = uuid.Nil }},
		{name: "empty batch", mutate: func(input *MatchReplayEventsIngestInput) { input.Events = nil }, wantErr: ErrReplayEventsEmpty},
		{name: "batch too large", mutate: func(input *MatchReplayEventsIngestInput) {
			input.Events = make([]MatchReplayEventIngestItem, replayEventsMaxBatchSize+1)
		}, wantErr: ErrReplayEventsBatchTooLarge},
		{name: "non participant", mutate: func(input *MatchReplayEventsIngestInput) { input.UserID = uuid.New() }, wantErr: ErrSubmissionUserNotInMatch},
		{name: "bad seq", mutate: func(input *MatchReplayEventsIngestInput) { input.Events[0].Seq = 0 }, wantErr: ErrReplayInvalidEvent},
		{name: "bad t_ms", mutate: func(input *MatchReplayEventsIngestInput) { input.Events[0].TMs = -1 }, wantErr: ErrReplayInvalidEvent},
		{name: "bad event type", mutate: func(input *MatchReplayEventsIngestInput) { input.Events[0].EventType = "paste" }, wantErr: ErrReplayInvalidEvent},
		{name: "bad json", mutate: func(input *MatchReplayEventsIngestInput) { input.Events[0].Payload = []byte("{") }, wantErr: ErrReplayInvalidEvent},
		{name: "payload too large", mutate: func(input *MatchReplayEventsIngestInput) {
			input.Events[0].Payload = []byte(`{"text":"` + strings.Repeat("x", replayEventPayloadMaxBytes) + `"}`)
		}, wantErr: ErrReplayEventPayloadTooLarge},
		{name: "seq not increasing", mutate: func(input *MatchReplayEventsIngestInput) {
			input.Events = append(input.Events, MatchReplayEventIngestItem{Seq: 1, TMs: 1, EventType: "cursor"})
		}, wantErr: ErrReplayInvalidEvent},
		{name: "time goes backwards", mutate: func(input *MatchReplayEventsIngestInput) {
			input.Events = append(input.Events, MatchReplayEventIngestItem{Seq: 2, TMs: -1, EventType: "cursor"})
		}, wantErr: ErrReplayInvalidEvent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := base
			input.Events = append([]MatchReplayEventIngestItem(nil), base.Events...)
			tt.mutate(&input)

			_, err := service.IngestMatchReplayEvents(context.Background(), input)
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestIngestMatchReplayEventsNormalizesAndStores(t *testing.T) {
	userID := uuid.New()
	matchID := uuid.New()
	repo := &fakeReplayMatchRepo{fakeMatchServiceRepo: fakeMatchServiceRepo{match: Match{
		ID:        matchID,
		Player1ID: userID,
		Player2ID: uuid.New(),
		Status:    MatchStatusRunning,
	}}}
	service := NewService(repo, nil, nil)

	result, err := service.IngestMatchReplayEvents(context.Background(), MatchReplayEventsIngestInput{
		MatchID: matchID,
		UserID:  userID,
		Events: []MatchReplayEventIngestItem{
			{Seq: 1, TMs: 0, EventType: " insert "},
			{Seq: 2, TMs: 0, EventType: "cursor", Payload: []byte(`{"line":1}`)},
		},
	})
	if err != nil {
		t.Fatalf("IngestMatchReplayEvents returned error: %v", err)
	}
	if result.Ingested != 2 || repo.eventsInput.Events[0].EventType != "insert" || string(repo.eventsInput.Events[0].Payload) != "{}" {
		t.Fatalf("result=%#v input=%#v", result, repo.eventsInput)
	}
}

func TestIngestMatchReplaySnapshotValidatesInput(t *testing.T) {
	player1 := uuid.New()
	matchID := uuid.New()
	repo := &fakeReplayMatchRepo{fakeMatchServiceRepo: fakeMatchServiceRepo{match: Match{
		ID:        matchID,
		Player1ID: player1,
		Player2ID: uuid.New(),
		Status:    MatchStatusRunning,
	}}}
	service := NewService(repo, nil, nil)

	base := MatchReplaySnapshotIngestInput{
		MatchID:    matchID,
		UserID:     player1,
		Seq:        1,
		TMs:        10,
		Language:   " go ",
		SourceCode: "package main",
	}
	tests := []struct {
		name    string
		mutate  func(*MatchReplaySnapshotIngestInput)
		wantErr error
	}{
		{name: "empty match", mutate: func(input *MatchReplaySnapshotIngestInput) { input.MatchID = uuid.Nil }},
		{name: "empty user", mutate: func(input *MatchReplaySnapshotIngestInput) { input.UserID = uuid.Nil }},
		{name: "bad seq", mutate: func(input *MatchReplaySnapshotIngestInput) { input.Seq = -1 }, wantErr: ErrReplayInvalidSnapshot},
		{name: "bad t_ms", mutate: func(input *MatchReplaySnapshotIngestInput) { input.TMs = -1 }, wantErr: ErrReplayInvalidSnapshot},
		{name: "initial t_ms non zero", mutate: func(input *MatchReplaySnapshotIngestInput) { input.Seq = 0; input.TMs = 1 }, wantErr: ErrReplayInvalidSnapshot},
		{name: "missing language", mutate: func(input *MatchReplaySnapshotIngestInput) { input.Language = " " }, wantErr: ErrReplayInvalidSnapshot},
		{name: "language too long", mutate: func(input *MatchReplaySnapshotIngestInput) {
			input.Language = strings.Repeat("x", replayLanguageMaxLength+1)
		}, wantErr: ErrReplayInvalidSnapshot},
		{name: "source too large", mutate: func(input *MatchReplaySnapshotIngestInput) {
			input.SourceCode = strings.Repeat("x", replaySnapshotSourceMaxBytes+1)
		}, wantErr: ErrReplaySnapshotTooLarge},
		{name: "non participant", mutate: func(input *MatchReplaySnapshotIngestInput) { input.UserID = uuid.New() }, wantErr: ErrSubmissionUserNotInMatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := base
			tt.mutate(&input)

			_, err := service.IngestMatchReplaySnapshot(context.Background(), input)
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestReplayCaptureAllowed(t *testing.T) {
	now := time.Now().UTC()
	finishedRecently := now.Add(-replayCaptureGracePeriod / 2)
	finishedLongAgo := now.Add(-2 * replayCaptureGracePeriod)

	tests := []struct {
		name  string
		match Match
		want  bool
	}{
		{name: "running", match: Match{Status: MatchStatusRunning}, want: true},
		{name: "finished in grace", match: Match{Status: MatchStatusFinished, FinishedAt: &finishedRecently}, want: true},
		{name: "cancelled in grace", match: Match{Status: MatchStatusCancelled, FinishedAt: &finishedRecently}, want: true},
		{name: "finished without timestamp", match: Match{Status: MatchStatusFinished}, want: false},
		{name: "finished after grace", match: Match{Status: MatchStatusFinished, FinishedAt: &finishedLongAgo}, want: false},
		{name: "pending", match: Match{Status: MatchStatusPending}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isReplayCaptureAllowed(tt.match, now); got != tt.want {
				t.Fatalf("isReplayCaptureAllowed = %v, want %v", got, tt.want)
			}
		})
	}
}
