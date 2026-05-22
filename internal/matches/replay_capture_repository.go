package matches

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const ensureReplaySessionQuery = `
	insert into match_replay_sessions (
		match_id,
		user_id,
		version,
		started_at,
		created_at,
		updated_at
	)
	values ($1, $2, $3, now(), now(), now())
	on conflict (match_id, user_id) do nothing
`

const getReplaySessionStateForUpdateQuery = `
	select
		last_seq,
		last_t_ms
	from match_replay_sessions
	where match_id = $1
	  and user_id = $2
	for update
`

const replayEventExistsQuery = `
	select exists(
		select 1
		from match_replay_events
		where match_id = $1
		  and user_id = $2
		  and seq = $3
	)
`

const replayEventExactDuplicateQuery = `
	select exists(
		select 1
		from match_replay_events
		where match_id = $1
		  and user_id = $2
		  and seq = $3
		  and t_ms = $4
		  and event_type = $5
		  and payload = $6::jsonb
	)
`

const insertReplayEventQuery = `
	insert into match_replay_events (
		match_id,
		user_id,
		seq,
		t_ms,
		event_type,
		payload,
		created_at
	)
	values ($1, $2, $3, $4, $5, $6, now())
`

const updateReplaySessionAfterEventsQuery = `
	update match_replay_sessions
	set
		last_seq = $3,
		last_t_ms = greatest(last_t_ms, $4),
		event_count = event_count + $5,
		duration_ms = greatest(duration_ms, $4),
		updated_at = now()
	where match_id = $1
	  and user_id = $2
`

const insertReplaySnapshotQuery = `
	insert into match_replay_snapshots (
		match_id,
		user_id,
		seq,
		t_ms,
		language,
		source_code,
		created_at
	)
	values ($1, $2, $3, $4, $5, $6, now())
	on conflict (match_id, user_id, seq) do nothing
`

const updateReplaySessionAfterSnapshotQuery = `
	update match_replay_sessions
	set
		snapshot_count = snapshot_count + $3,
		updated_at = now()
	where match_id = $1
	  and user_id = $2
`

const listMatchReplayTimelineEventsQuery = `
	select
		match_id,
		user_id,
		seq,
		t_ms,
		event_type,
		payload,
		created_at
	from match_replay_events
	where match_id = $1
	order by t_ms asc, seq asc, id asc
`

const listMatchReplayTimelineSnapshotsQuery = `
	select
		match_id,
		user_id,
		seq,
		t_ms,
		language,
		source_code,
		created_at
	from match_replay_snapshots
	where match_id = $1
	order by t_ms asc, seq asc, id asc
`

func (r *Repository) IngestMatchReplayEvents(ctx context.Context, input MatchReplayEventsIngestInput) (MatchReplayEventsIngestResult, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MatchReplayEventsIngestResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, ensureReplaySessionQuery, input.MatchID, input.UserID, replayCaptureTimelineVersion); err != nil {
		return MatchReplayEventsIngestResult{}, fmt.Errorf("ensure replay session: %w", err)
	}

	var lastSeq int64
	var lastTMs int64
	if err := tx.QueryRow(ctx, getReplaySessionStateForUpdateQuery, input.MatchID, input.UserID).Scan(&lastSeq, &lastTMs); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MatchReplayEventsIngestResult{}, ErrMatchNotFound
		}
		return MatchReplayEventsIngestResult{}, fmt.Errorf("get replay session state: %w", err)
	}

	expectedSeq := lastSeq + 1
	ingested := 0
	deduplicated := 0
	latestTMs := lastTMs

	for _, event := range input.Events {
		switch {
		case event.Seq < expectedSeq:
			var exactDuplicate bool
			if err := tx.QueryRow(
				ctx,
				replayEventExactDuplicateQuery,
				input.MatchID,
				input.UserID,
				event.Seq,
				event.TMs,
				event.EventType,
				event.Payload,
			).Scan(&exactDuplicate); err != nil {
				return MatchReplayEventsIngestResult{}, fmt.Errorf("check replay event duplicate: %w", err)
			}
			if exactDuplicate {
				deduplicated++
				continue
			}

			var exists bool
			if err := tx.QueryRow(ctx, replayEventExistsQuery, input.MatchID, input.UserID, event.Seq).Scan(&exists); err != nil {
				return MatchReplayEventsIngestResult{}, fmt.Errorf("check replay event seq existence: %w", err)
			}
			if exists {
				return MatchReplayEventsIngestResult{}, fmt.Errorf("%w: conflicting replay event payload for seq %d", ErrReplaySeqConflict, event.Seq)
			}

			return MatchReplayEventsIngestResult{}, fmt.Errorf("%w: expected seq %d", ErrReplaySeqConflict, expectedSeq)

		case event.Seq > expectedSeq:
			return MatchReplayEventsIngestResult{}, fmt.Errorf("%w: expected seq %d", ErrReplaySeqConflict, expectedSeq)
		}

		if event.TMs < latestTMs {
			return MatchReplayEventsIngestResult{}, fmt.Errorf("%w: t_ms must be >= last_t_ms", ErrReplaySeqConflict)
		}

		if _, err := tx.Exec(ctx, insertReplayEventQuery,
			input.MatchID,
			input.UserID,
			event.Seq,
			event.TMs,
			event.EventType,
			event.Payload,
		); err != nil {
			return MatchReplayEventsIngestResult{}, fmt.Errorf("insert replay event: %w", err)
		}

		ingested++
		expectedSeq++
		if event.TMs > latestTMs {
			latestTMs = event.TMs
		}
	}

	if ingested > 0 {
		lastSeq = expectedSeq - 1
		if _, err := tx.Exec(ctx, updateReplaySessionAfterEventsQuery,
			input.MatchID,
			input.UserID,
			lastSeq,
			latestTMs,
			ingested,
		); err != nil {
			return MatchReplayEventsIngestResult{}, fmt.Errorf("update replay session after events: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return MatchReplayEventsIngestResult{}, fmt.Errorf("commit tx: %w", err)
	}

	return MatchReplayEventsIngestResult{
		Ingested:     ingested,
		Deduplicated: deduplicated,
		LastSeq:      lastSeq,
		ExpectedSeq:  lastSeq + 1,
	}, nil
}

func (r *Repository) IngestMatchReplaySnapshot(ctx context.Context, input MatchReplaySnapshotIngestInput) (MatchReplaySnapshotIngestResult, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MatchReplaySnapshotIngestResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, ensureReplaySessionQuery, input.MatchID, input.UserID, replayCaptureTimelineVersion); err != nil {
		return MatchReplaySnapshotIngestResult{}, fmt.Errorf("ensure replay session: %w", err)
	}

	var lastSeq int64
	var ignoredLastTMs int64
	if err := tx.QueryRow(ctx, getReplaySessionStateForUpdateQuery, input.MatchID, input.UserID).Scan(&lastSeq, &ignoredLastTMs); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MatchReplaySnapshotIngestResult{}, ErrMatchNotFound
		}
		return MatchReplaySnapshotIngestResult{}, fmt.Errorf("get replay session state: %w", err)
	}

	if input.Seq > lastSeq {
		return MatchReplaySnapshotIngestResult{}, fmt.Errorf("%w: expected seq <= %d", ErrReplaySnapshotAheadOfEvents, lastSeq)
	}

	ct, err := tx.Exec(ctx, insertReplaySnapshotQuery,
		input.MatchID,
		input.UserID,
		input.Seq,
		input.TMs,
		input.Language,
		input.SourceCode,
	)
	if err != nil {
		return MatchReplaySnapshotIngestResult{}, fmt.Errorf("insert replay snapshot: %w", err)
	}

	inserted := ct.RowsAffected() > 0
	snapshotDelta := 0
	if inserted {
		snapshotDelta = 1
	}

	if snapshotDelta > 0 {
		if _, err := tx.Exec(ctx, updateReplaySessionAfterSnapshotQuery,
			input.MatchID,
			input.UserID,
			snapshotDelta,
		); err != nil {
			return MatchReplaySnapshotIngestResult{}, fmt.Errorf("update replay session after snapshot: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return MatchReplaySnapshotIngestResult{}, fmt.Errorf("commit tx: %w", err)
	}

	return MatchReplaySnapshotIngestResult{
		Inserted:    inserted,
		LastSeq:     lastSeq,
		ExpectedSeq: lastSeq + 1,
	}, nil
}

func (r *Repository) ListMatchReplayTimelineEvents(ctx context.Context, matchID uuid.UUID) ([]MatchReplayCapturedEvent, error) {
	rows, err := r.db.Query(ctx, listMatchReplayTimelineEventsQuery, matchID)
	if err != nil {
		return nil, fmt.Errorf("list replay timeline events: %w", err)
	}
	defer rows.Close()

	items := make([]MatchReplayCapturedEvent, 0)
	for rows.Next() {
		var item MatchReplayCapturedEvent
		if err := rows.Scan(
			&item.MatchID,
			&item.UserID,
			&item.Seq,
			&item.TMs,
			&item.EventType,
			&item.Payload,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan replay timeline event: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate replay timeline events: %w", err)
	}

	return items, nil
}

func (r *Repository) ListMatchReplayTimelineSnapshots(ctx context.Context, matchID uuid.UUID) ([]MatchReplayCapturedSnapshot, error) {
	rows, err := r.db.Query(ctx, listMatchReplayTimelineSnapshotsQuery, matchID)
	if err != nil {
		return nil, fmt.Errorf("list replay timeline snapshots: %w", err)
	}
	defer rows.Close()

	items := make([]MatchReplayCapturedSnapshot, 0)
	for rows.Next() {
		var item MatchReplayCapturedSnapshot
		if err := rows.Scan(
			&item.MatchID,
			&item.UserID,
			&item.Seq,
			&item.TMs,
			&item.Language,
			&item.SourceCode,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan replay timeline snapshot: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate replay timeline snapshots: %w", err)
	}

	return items, nil
}
