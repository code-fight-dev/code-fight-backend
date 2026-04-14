package matches

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type SubmissionReader interface {
	GetSubmissionForMatchProcessing(ctx context.Context, submissionID uuid.UUID) (SubmissionMatchView, error)
}

type SubmissionMatchView struct {
	ID         uuid.UUID
	MatchID    *uuid.UUID
	TaskID     uuid.UUID
	UserID     uuid.UUID
	Status     string
	Verdict    *string
	Score      int
	FinishedAt *time.Time
}