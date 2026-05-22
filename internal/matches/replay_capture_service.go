package matches

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var allowedReplayEventTypes = map[string]struct{}{
	"insert":          {},
	"delete":          {},
	"replace":         {},
	"cursor":          {},
	"selection":       {},
	"language_switch": {},
}

type replayCaptureRepository interface {
	IngestMatchReplayEvents(ctx context.Context, input MatchReplayEventsIngestInput) (MatchReplayEventsIngestResult, error)
	IngestMatchReplaySnapshot(ctx context.Context, input MatchReplaySnapshotIngestInput) (MatchReplaySnapshotIngestResult, error)
}

func (s *Service) IngestMatchReplayEvents(ctx context.Context, input MatchReplayEventsIngestInput) (MatchReplayEventsIngestResult, error) {
	if input.MatchID == uuid.Nil {
		return MatchReplayEventsIngestResult{}, fmt.Errorf("ingest replay events: empty match id")
	}
	if input.UserID == uuid.Nil {
		return MatchReplayEventsIngestResult{}, fmt.Errorf("ingest replay events: empty user id")
	}
	if len(input.Events) == 0 {
		return MatchReplayEventsIngestResult{}, ErrReplayEventsEmpty
	}
	if len(input.Events) > replayEventsMaxBatchSize {
		return MatchReplayEventsIngestResult{}, ErrReplayEventsBatchTooLarge
	}

	match, err := s.repo.GetMatch(ctx, input.MatchID)
	if err != nil {
		return MatchReplayEventsIngestResult{}, err
	}
	if input.UserID != match.Player1ID && input.UserID != match.Player2ID {
		return MatchReplayEventsIngestResult{}, ErrSubmissionUserNotInMatch
	}
	if !isReplayCaptureAllowed(match, time.Now().UTC()) {
		return MatchReplayEventsIngestResult{}, ErrMatchReplayCaptureUnavailable
	}

	var previousSeq int64
	var previousTMs int64
	for index, event := range input.Events {
		if event.Seq <= 0 {
			return MatchReplayEventsIngestResult{}, fmt.Errorf("%w: seq must be > 0", ErrReplayInvalidEvent)
		}
		if event.TMs < 0 {
			return MatchReplayEventsIngestResult{}, fmt.Errorf("%w: t_ms must be >= 0", ErrReplayInvalidEvent)
		}

		eventType := strings.TrimSpace(event.EventType)
		if _, ok := allowedReplayEventTypes[eventType]; !ok {
			return MatchReplayEventsIngestResult{}, fmt.Errorf("%w: unsupported event_type", ErrReplayInvalidEvent)
		}
		input.Events[index].EventType = eventType

		if len(event.Payload) == 0 {
			input.Events[index].Payload = json.RawMessage(`{}`)
		} else if len(event.Payload) > replayEventPayloadMaxBytes {
			return MatchReplayEventsIngestResult{}, ErrReplayEventPayloadTooLarge
		} else if !json.Valid(event.Payload) {
			return MatchReplayEventsIngestResult{}, fmt.Errorf("%w: payload must be valid json", ErrReplayInvalidEvent)
		}

		if index > 0 {
			if event.Seq <= previousSeq {
				return MatchReplayEventsIngestResult{}, fmt.Errorf("%w: batch seq must be strictly increasing", ErrReplayInvalidEvent)
			}
			if event.TMs < previousTMs {
				return MatchReplayEventsIngestResult{}, fmt.Errorf("%w: batch t_ms must be non-decreasing", ErrReplayInvalidEvent)
			}
		}

		previousSeq = event.Seq
		previousTMs = event.TMs
	}

	repo, ok := s.repo.(replayCaptureRepository)
	if !ok {
		return MatchReplayEventsIngestResult{}, fmt.Errorf("ingest replay events: repository does not support replay capture")
	}

	return repo.IngestMatchReplayEvents(ctx, input)
}

func (s *Service) IngestMatchReplaySnapshot(ctx context.Context, input MatchReplaySnapshotIngestInput) (MatchReplaySnapshotIngestResult, error) {
	if input.MatchID == uuid.Nil {
		return MatchReplaySnapshotIngestResult{}, fmt.Errorf("ingest replay snapshot: empty match id")
	}
	if input.UserID == uuid.Nil {
		return MatchReplaySnapshotIngestResult{}, fmt.Errorf("ingest replay snapshot: empty user id")
	}
	if input.Seq < 0 {
		return MatchReplaySnapshotIngestResult{}, fmt.Errorf("%w: seq must be >= 0", ErrReplayInvalidSnapshot)
	}
	if input.TMs < 0 {
		return MatchReplaySnapshotIngestResult{}, fmt.Errorf("%w: t_ms must be >= 0", ErrReplayInvalidSnapshot)
	}
	if input.Seq == 0 && input.TMs != 0 {
		return MatchReplaySnapshotIngestResult{}, fmt.Errorf("%w: initial snapshot t_ms must be 0", ErrReplayInvalidSnapshot)
	}

	input.Language = strings.TrimSpace(input.Language)
	if input.Language == "" {
		return MatchReplaySnapshotIngestResult{}, fmt.Errorf("%w: language is required", ErrReplayInvalidSnapshot)
	}
	if len(input.Language) > replayLanguageMaxLength {
		return MatchReplaySnapshotIngestResult{}, fmt.Errorf("%w: language is too long", ErrReplayInvalidSnapshot)
	}
	if len(input.SourceCode) > replaySnapshotSourceMaxBytes {
		return MatchReplaySnapshotIngestResult{}, ErrReplaySnapshotTooLarge
	}

	match, err := s.repo.GetMatch(ctx, input.MatchID)
	if err != nil {
		return MatchReplaySnapshotIngestResult{}, err
	}
	if input.UserID != match.Player1ID && input.UserID != match.Player2ID {
		return MatchReplaySnapshotIngestResult{}, ErrSubmissionUserNotInMatch
	}
	if !isReplayCaptureAllowed(match, time.Now().UTC()) {
		return MatchReplaySnapshotIngestResult{}, ErrMatchReplayCaptureUnavailable
	}

	repo, ok := s.repo.(replayCaptureRepository)
	if !ok {
		return MatchReplaySnapshotIngestResult{}, fmt.Errorf("ingest replay snapshot: repository does not support replay capture")
	}

	return repo.IngestMatchReplaySnapshot(ctx, input)
}

type replayTimelineRepository interface {
	ListMatchReplayTimelineEvents(ctx context.Context, matchID uuid.UUID) ([]MatchReplayCapturedEvent, error)
	ListMatchReplayTimelineSnapshots(ctx context.Context, matchID uuid.UUID) ([]MatchReplayCapturedSnapshot, error)
}

func (s *Service) loadCapturedReplayTimeline(ctx context.Context, matchID uuid.UUID) ([]MatchReplayEvent, []MatchReplaySnapshot, error) {
	repo, ok := s.repo.(replayTimelineRepository)
	if !ok {
		return nil, nil, fmt.Errorf("repository does not support replay timeline")
	}

	eventRows, err := repo.ListMatchReplayTimelineEvents(ctx, matchID)
	if err != nil {
		return nil, nil, err
	}
	snapshotRows, err := repo.ListMatchReplayTimelineSnapshots(ctx, matchID)
	if err != nil {
		return nil, nil, err
	}

	events := make([]MatchReplayEvent, 0, len(eventRows))
	for _, item := range eventRows {
		events = append(events, MatchReplayEvent{
			UserID:  item.UserID,
			Seq:     item.Seq,
			TMs:     item.TMs,
			Type:    item.EventType,
			Payload: item.Payload,
		})
	}

	snapshots := make([]MatchReplaySnapshot, 0, len(snapshotRows))
	for _, item := range snapshotRows {
		snapshots = append(snapshots, MatchReplaySnapshot{
			UserID:     item.UserID,
			Seq:        item.Seq,
			TMs:        item.TMs,
			Language:   item.Language,
			SourceCode: item.SourceCode,
		})
	}

	return events, snapshots, nil
}

func isReplayCaptureAllowed(match Match, now time.Time) bool {
	switch match.Status {
	case MatchStatusRunning:
		return true
	case MatchStatusFinished, MatchStatusCancelled:
		if match.FinishedAt == nil || match.FinishedAt.IsZero() {
			return false
		}
		return !now.After(match.FinishedAt.UTC().Add(replayCaptureGracePeriod))
	default:
		return false
	}
}
