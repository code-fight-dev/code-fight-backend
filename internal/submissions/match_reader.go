package submissions

import (
	"context"

	"github.com/google/uuid"

	"code-fight-backend/internal/matches"
)

type MatchReader struct {
	repo Repository
}

func NewMatchReader(repo Repository) *MatchReader {
	return &MatchReader{repo: repo}
}

func (r *MatchReader) GetSubmissionForMatchProcessing(ctx context.Context, submissionID uuid.UUID) (matches.SubmissionMatchView, error) {
	sub, _, err := r.repo.GetByID(ctx, submissionID.String())
	if err != nil {
		return matches.SubmissionMatchView{}, err
	}

	var matchID *uuid.UUID
	if sub.MatchID != nil && *sub.MatchID != "" {
		parsedMatchID, err := uuid.Parse(*sub.MatchID)
		if err != nil {
			return matches.SubmissionMatchView{}, err
		}
		matchID = &parsedMatchID
	}

	taskID, err := uuid.Parse(sub.TaskID)
	if err != nil {
		return matches.SubmissionMatchView{}, err
	}

	userID, err := uuid.Parse(sub.UserID)
	if err != nil {
		return matches.SubmissionMatchView{}, err
	}

	return matches.SubmissionMatchView{
		ID:         submissionID,
		MatchID:    matchID,
		TaskID:     taskID,
		UserID:     userID,
		Status:     sub.Status,
		Verdict:    sub.Verdict,
		Score:      sub.Score,
		FinishedAt: sub.FinishedAt,
	}, nil
}