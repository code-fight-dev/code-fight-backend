package tournament

import (
	"errors"
	"testing"
	"time"

	"code-fight-backend/internal/access"
	"code-fight-backend/internal/judge"
	"code-fight-backend/internal/notifications"
	"code-fight-backend/internal/submissions"
	"code-fight-backend/internal/testsupport"

	"github.com/google/uuid"
)

func TestSharedProblemContestLifecycle(t *testing.T) {
	db, ctx := testsupport.RequirePostgres(t)
	service := NewService(db)
	organizer := createTournamentPlayer(t, db, ctx, access.Organizer)
	first := createTournamentPlayer(t, db, ctx, access.User)
	second := createTournamentPlayer(t, db, ctx, access.User)
	third := createTournamentPlayer(t, db, ctx, access.User)
	noShow := createTournamentPlayer(t, db, ctx, access.User)
	draft, err := service.CreateDraft(ctx, organizer, CreateInput{Slug: "shared-" + uuid.NewString(), Title: "Shared cup"}, "shared-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, "delete from submissions where tournament_id=$1", draft.ID)
		_, _ = db.Exec(ctx, "delete from tournaments where id=$1", draft.ID)
	})
	now := time.Now().UTC().Truncate(time.Second)
	closes, starts, ends := now.Add(time.Minute), now.Add(2*time.Minute), now.Add(5*time.Minute)
	publish := TransitionInput{
		Status: "published", Format: sharedProblemSet, ExpectedConfigVersion: 1,
		TaskSlugs: []string{"mirror-log", "missing-problem"},
		Schedule:  Schedule{RegistrationClosesAt: &closes, StartsAt: &starts, EndsAt: &ends},
	}
	if err := service.Transition(ctx, organizer, draft.ID, publish, "shared-test"); !errors.Is(err, access.ErrInvalidInput) {
		t.Fatalf("publish with a missing task: %v", err)
	}
	publish.TaskSlugs = []string{"mirror-log", "signal-stabilizer"}
	if err := service.Transition(ctx, organizer, draft.ID, publish, "shared-test"); err != nil {
		t.Fatal(err)
	}
	for _, player := range []uuid.UUID{first, second, third, noShow, first} {
		if err := service.Join(ctx, player, draft.ID); err != nil {
			t.Fatal(err)
		}
	}
	board, err := service.Board(ctx, draft.ID)
	if err != nil || board.Tournament.Format != sharedProblemSet || board.Tournament.TaskID != nil ||
		board.Problems == nil || len(*board.Problems) != 0 || len(board.Standings) != 4 {
		t.Fatalf("pre-start board = %#v, %v", board, err)
	}

	var mirror, outside uuid.UUID
	if err := db.QueryRow(ctx, `select a.id,b.id from tasks a cross join tasks b
		where a.slug='mirror-log' and b.slug='cache-key-collisions'`).Scan(&mirror, &outside); err != nil {
		t.Fatal(err)
	}
	subService := submissions.NewService(submissions.NewPGRepository(db), submissions.NewPGTaskRepository(db), nil, nil, nil, "", "", 65536, nil)
	code := submissions.CreateSubmissionRequest{Language: "go", LanguageVersion: "1.24", SourceCode: "package main\nfunc main() {}"}
	if _, err := subService.CreateContestSubmission(ctx, first, draft.ID, mirror, code); !errors.Is(err, submissions.ErrContestSubmissionWindow) {
		t.Fatalf("early submission = %v", err)
	}

	starts = now.Add(-time.Hour)
	if _, err := db.Exec(ctx, `update tournaments set registration_closes_at=$2, starts_at=$2 where id=$1`, draft.ID, starts); err != nil {
		t.Fatal(err)
	}
	if advanced, err := service.advanceSchedule(ctx); !advanced || err != nil {
		t.Fatalf("start contest: %v, %v", advanced, err)
	}
	board, err = service.Board(ctx, draft.ID)
	if err != nil || board.Tournament.Status != "running" || len(board.Fixtures) != 0 ||
		board.Problems == nil || len(*board.Problems) != 2 || (*board.Problems)[0].TaskID != mirror ||
		(*board.Problems)[1].Slug != "signal-stabilizer" {
		t.Fatalf("running board = %#v, %v", board, err)
	}
	if err := service.Join(ctx, organizer, draft.ID); !errors.Is(err, access.ErrConflict) {
		t.Fatalf("late registration = %v", err)
	}
	if _, err := subService.CreateContestSubmission(ctx, organizer, draft.ID, mirror, code); !errors.Is(err, submissions.ErrContestNotJoined) {
		t.Fatalf("unregistered submission = %v", err)
	}
	if _, err := subService.CreateContestSubmission(ctx, first, draft.ID, outside, code); !errors.Is(err, submissions.ErrContestProblemNotFound) {
		t.Fatalf("unassigned problem = %v", err)
	}

	submit := func(player uuid.UUID, offset time.Duration) *submissions.Submission {
		t.Helper()
		sub, err := subService.CreateContestSubmission(ctx, player, draft.ID, mirror, code)
		if err != nil || sub.TournamentID == nil || *sub.TournamentID != draft.ID.String() || sub.Status != submissions.StatusQueued {
			t.Fatalf("queued contest submission = %#v, %v", sub, err)
		}
		if _, err := db.Exec(ctx, "update submissions set created_at=$2 where id=$1", sub.ID, starts.Add(offset)); err != nil {
			t.Fatal(err)
		}
		return sub
	}
	settle := func(sub *submissions.Submission, verdict string) {
		t.Helper()
		started := now.Add(2 * time.Second)
		finished := now.Add(3 * time.Second)
		if err := subService.HandleJudgeCallback(ctx, judge.CallbackPayload{
			SubmissionID: sub.ID, JudgeSubmissionID: sub.ID, JudgeStatus: submissions.JudgeStatusFinished,
			Verdict: &verdict, StartedAt: &started, FinishedAt: &finished,
		}); err != nil {
			t.Fatalf("persist verdict %s: %v", verdict, err)
		}
	}

	wrong := submit(first, 5*time.Second)
	compile := submit(first, 10*time.Second)
	firstSolve := submit(first, 15*time.Second)
	laterSolve := submit(first, 30*time.Second)
	secondSolve := submit(second, 1215*time.Second)
	thirdWrong := submit(third, 40*time.Second)
	var queued bool
	if err := db.QueryRow(ctx, `select exists (select 1 from judge_dispatch where submission_id=$1)`, firstSolve.ID).Scan(&queued); err != nil || !queued {
		t.Fatalf("durable dispatch = %v, %v", queued, err)
	}
	if _, _, err := subService.GetByIDForUser(ctx, firstSolve.ID, second); !errors.Is(err, submissions.ErrSubmissionNotFound) {
		t.Fatalf("another participant can read the submission: %v", err)
	}
	settle(laterSolve, submissions.VerdictAccepted)
	settle(compile, submissions.VerdictCompileError)
	settle(wrong, submissions.VerdictWrongAnswer)
	settle(secondSolve, submissions.VerdictAccepted)
	settle(thirdWrong, submissions.VerdictWrongAnswer)

	if _, err := db.Exec(ctx, "update tournaments set ends_at=$2 where id=$1", draft.ID, now.Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	if advanced, err := service.advanceSchedule(ctx); advanced || err != nil {
		t.Fatalf("advanced with an unsettled submission: %v, %v", advanced, err)
	}
	if _, err := subService.CreateContestSubmission(ctx, first, draft.ID, mirror, code); !errors.Is(err, submissions.ErrContestSubmissionWindow) {
		t.Fatalf("post-deadline submission = %v", err)
	}
	settle(firstSolve, submissions.VerdictAccepted)
	if advanced, err := service.advanceSchedule(ctx); !advanced || err != nil {
		t.Fatalf("complete after verdict: %v, %v", advanced, err)
	}
	settle(firstSolve, submissions.VerdictAccepted)
	board, err = service.Board(ctx, draft.ID)
	if err != nil || board.Tournament.Status != "completed" || len(board.Standings) != 4 {
		t.Fatalf("final board = %#v, %v", board, err)
	}
	for _, standing := range board.Standings {
		switch standing.UserID {
		case first, second:
			if standing.Rank != 1 || standing.Solved == nil || *standing.Solved != 1 ||
				standing.PenaltySeconds == nil || *standing.PenaltySeconds != 1215 ||
				standing.ContestRatingAfter == nil || *standing.ContestRatingAfter != 1008 {
				t.Fatalf("tied first place = %#v", standing)
			}
		case third:
			if standing.Rank != 2 || standing.ContestRatingAfter == nil || *standing.ContestRatingAfter != 984 {
				t.Fatalf("rated participant with zero solves = %#v", standing)
			}
		case noShow:
			if standing.Rank != 2 || standing.ContestRatingAfter != nil {
				t.Fatalf("unrated participant with no submissions = %#v", standing)
			}
		}
	}
	history, err := service.History(ctx, first, 10, "")
	if err != nil || len(history.Items) != 1 {
		t.Fatalf("participation history = %#v, %v", history, err)
	}
	item := history.Items[0]
	if item.Format != sharedProblemSet || item.CompletedAt == nil || item.Rank == nil || *item.Rank != 1 ||
		item.Points != 1 || item.PenaltySeconds == nil || *item.PenaltySeconds != 1215 ||
		item.ContestRatingAfter == nil || *item.ContestRatingAfter != 1008 {
		t.Fatalf("persisted contest result = %#v", item)
	}
	inbox, err := notifications.NewStore(db).List(ctx, first, nil)
	if err != nil || len(inbox.Items) != 2 || inbox.Items[0].Kind != "tournament.completed" {
		t.Fatalf("contest notifications = %#v, %v", inbox, err)
	}
}
