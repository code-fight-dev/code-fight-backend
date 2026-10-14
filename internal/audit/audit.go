package audit

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Event struct {
	ActorID      *uuid.UUID
	Action       string
	EntityType   string
	EntityID     uuid.UUID
	TournamentID *uuid.UUID
	RequestID    string
	Metadata     map[string]any
}

func Append(ctx context.Context, tx pgx.Tx, event Event) error {
	if event.Action == "" || event.EntityType == "" || event.EntityID == uuid.Nil || event.RequestID == "" {
		return errors.New("invalid audit event")
	}
	actorType := "system"
	if event.ActorID != nil {
		if *event.ActorID == uuid.Nil {
			return errors.New("invalid audit actor")
		}
		actorType = "user"
	}
	if event.Metadata == nil {
		event.Metadata = map[string]any{}
	}
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `insert into audit_events
        (id, actor_type, actor_id, action, entity_type, entity_id, tournament_id, request_id, metadata)
        values ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb)`,
		uuid.New(), actorType, event.ActorID, event.Action, event.EntityType,
		event.EntityID, event.TournamentID, event.RequestID, string(metadata))
	return err
}
