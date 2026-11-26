package submissions

import (
	"context"
	"errors"
	"testing"
	"time"

	"code-fight-backend/internal/judge"
	"code-fight-backend/internal/matches"
	"code-fight-backend/internal/testsupport"

	"github.com/google/uuid"
)

func TestPGJudgeCallbackAppliesMatchProgressOnce(t *testing.T) {
	for _, scenario := range []string{"sequential deliveries", "concurrent deliveries", "finalization retry"} {
		t.Run(scenario, func(t *testing.T) {
			db, ctx := testsupport.RequirePostgres(t)
			repo := NewPGRepository(db)
			matchRepo := matches.NewRepository(db)
			player1 := createSubmissionTestUser(t, db, ctx)
			player2 := createSubmissionTestUser(t, db, ctx)
			taskID, sampleID, _ := createSubmissionTestTask(t, db, ctx)
			matchID := uuid.New()
			if _, err := matchRepo.CreateMatch(ctx, matches.Match{
				ID: matchID, Player1ID: player1, Player2ID: player2, TaskID: &taskID,
				TaskMode: matches.TaskModeNormal, RatingMode: "global",
				Status: matches.MatchStatusPending, JudgeStatus: matches.JudgeStatusNotStarted,
			}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_, _ = db.Exec(ctx, "delete from submissions where match_id = $1", matchID)
				_, _ = db.Exec(ctx, "delete from matches where id = $1", matchID)
			})
			for _, player := range []uuid.UUID{player1, player2} {
				if _, err := matchRepo.StartMatch(ctx, matchID, player, time.Minute); err != nil {
					t.Fatal(err)
				}
			}
			now := time.Now().UTC().Truncate(time.Microsecond)
			id, judgeID, matchIDString := uuid.NewString(), uuid.NewString(), matchID.String()
			if err := repo.Create(ctx, &Submission{
				ID: id, JudgeSubmissionID: &judgeID, MatchID: &matchIDString,
				TaskID: taskID.String(), UserID: player1.String(), Status: StatusSentToJudge,
				Language: "go", LanguageVersion: "1.24", SourceCode: "package main", CreatedAt: now, UpdatedAt: now,
			}); err != nil {
				t.Fatal(err)
			}
			verdict := VerdictWrongAnswer
			if scenario == "finalization retry" {
				verdict = VerdictAccepted
			}
			payload := judge.CallbackPayload{
				SubmissionID: id, JudgeSubmissionID: judgeID, JudgeStatus: JudgeStatusFinished,
				Verdict: &verdict, StartedAt: &now, FinishedAt: &now, TotalTests: 1,
				Tests: []judge.CallbackTestResult{{TestCaseID: sampleID.String(), Status: verdict}},
			}
			interrupted := &interruptMatchFinalization{Repository: matchRepo}
			matchService := matches.NewService(interrupted, nil, NewMatchReader(repo))
			service := NewService(repo, nil, nil, nil, matchService, "", "", 0, nil)
			switch scenario {
			case "concurrent deliveries":
				const deliveries = 8
				start := make(chan struct{})
				results := make(chan error, deliveries)
				for range deliveries {
					go func() {
						<-start
						results <- service.HandleJudgeCallback(ctx, payload)
					}()
				}
				close(start)
				for range deliveries {
					if err := <-results; err != nil {
						t.Errorf("concurrent callback: %v", err)
					}
				}
			case "finalization retry":
				interrupted.failure = errors.New("match finalization interrupted")
				if err := service.HandleJudgeCallback(ctx, payload); !errors.Is(err, interrupted.failure) {
					t.Fatalf("first callback returned %v", err)
				}
				interrupted.failure = nil
				pending, err := matchRepo.GetMatch(ctx, matchID)
				if err != nil || pending.Player1Attempts != 1 || pending.Status != matches.MatchStatusRunning {
					t.Fatalf("interrupted finalization persisted progress once: %#v, %v", pending, err)
				}
				if err := service.HandleJudgeCallback(ctx, payload); err != nil {
					t.Fatalf("retry finalization: %v", err)
				}
			default:
				for range 2 {
					if err := service.HandleJudgeCallback(ctx, payload); err != nil {
						t.Fatal(err)
					}
				}
			}
			wantStatus := matches.MatchStatusRunning
			if verdict == VerdictAccepted {
				wantStatus = matches.MatchStatusFinished
			}
			final, err := matchRepo.GetMatch(ctx, matchID)
			if err != nil || final.Player1Attempts != 1 || final.Player2Attempts != 0 || final.Status != wantStatus {
				t.Fatalf("match after callbacks: %#v, %v", final, err)
			}
		})
	}
}

type interruptMatchFinalization struct {
	*matches.Repository
	failure error
}

func (r *interruptMatchFinalization) FinishMatch(ctx context.Context, input matches.FinishMatchInput) error {
	if r.failure != nil {
		return r.failure
	}
	return r.Repository.FinishMatch(ctx, input)
}
