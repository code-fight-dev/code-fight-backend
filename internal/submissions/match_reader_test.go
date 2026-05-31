package submissions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMatchReaderGetSubmissionForMatchProcessing(t *testing.T) {
	ctx := context.Background()
	submissionID := uuid.New()
	matchID := uuid.New()
	taskID := uuid.New()
	userID := uuid.New()
	verdict := VerdictAccepted
	finishedAt := time.Now().UTC()
	repo := submissionRepoFixture()
	repo.submissions = map[string]*Submission{
		submissionID.String(): {
			ID:         submissionID.String(),
			MatchID:    ptrString(matchID.String()),
			TaskID:     taskID.String(),
			UserID:     userID.String(),
			Status:     StatusFinished,
			Verdict:    &verdict,
			Score:      100,
			FinishedAt: &finishedAt,
		},
	}

	got, err := NewMatchReader(repo).GetSubmissionForMatchProcessing(ctx, submissionID)
	if err != nil {
		t.Fatalf("GetSubmissionForMatchProcessing returned error: %v", err)
	}

	if got.ID != submissionID {
		t.Fatalf("id = %s, want %s", got.ID, submissionID)
	}
	if got.MatchID == nil || *got.MatchID != matchID {
		t.Fatalf("match id = %v, want %s", got.MatchID, matchID)
	}
	if got.TaskID != taskID {
		t.Fatalf("task id = %s, want %s", got.TaskID, taskID)
	}
	if got.UserID != userID {
		t.Fatalf("user id = %s, want %s", got.UserID, userID)
	}
	if got.Status != StatusFinished || got.Verdict == nil || *got.Verdict != VerdictAccepted || got.Score != 100 {
		t.Fatalf("match view = %#v", got)
	}
	if got.FinishedAt == nil || !got.FinishedAt.Equal(finishedAt) {
		t.Fatalf("finished at = %v, want %v", got.FinishedAt, finishedAt)
	}
}

func TestMatchReaderGetSubmissionForMatchProcessingInvalidIDs(t *testing.T) {
	ctx := context.Background()
	submissionID := uuid.New()
	validID := uuid.NewString()

	tests := []struct {
		name string
		sub  *Submission
	}{
		{
			name: "match id",
			sub:  &Submission{ID: submissionID.String(), MatchID: ptrString("bad-match-id"), TaskID: validID, UserID: validID},
		},
		{
			name: "task id",
			sub:  &Submission{ID: submissionID.String(), TaskID: "bad-task-id", UserID: validID},
		},
		{
			name: "user id",
			sub:  &Submission{ID: submissionID.String(), TaskID: validID, UserID: "bad-user-id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := submissionRepoFixture()
			repo.submissions = map[string]*Submission{submissionID.String(): tt.sub}

			if _, err := NewMatchReader(repo).GetSubmissionForMatchProcessing(ctx, submissionID); err == nil {
				t.Fatal("expected invalid id error")
			}
		})
	}
}

func TestRepositoryHelpers(t *testing.T) {
	repo := NewPGRepository(nil)
	if repo == nil {
		t.Fatal("NewPGRepository returned nil")
	}

	duplicateErr := errors.New("violates submissions_judge_submission_id_uidx")
	if err := mapPQError(duplicateErr); !errors.Is(err, ErrDuplicateJudgeSubmissionID) {
		t.Fatalf("duplicate error = %v, want %v", err, ErrDuplicateJudgeSubmissionID)
	}

	otherErr := errors.New("other db error")
	if err := mapPQError(otherErr); !errors.Is(err, otherErr) {
		t.Fatalf("other error = %v, want original", err)
	}

	if !contains("abcdef", "bcd") {
		t.Fatal("contains returned false for existing substring")
	}
	if contains("abc", "") {
		t.Fatal("contains returned true for empty substring")
	}
	if !stringContains("abc", "") {
		t.Fatal("stringContains returned false for empty substring")
	}
	if got := indexOf("abc", "d"); got != -1 {
		t.Fatalf("indexOf = %d, want -1", got)
	}
}
