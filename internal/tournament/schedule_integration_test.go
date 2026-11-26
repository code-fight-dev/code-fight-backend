package tournament

import (
	"errors"
	"testing"
	"time"

	"code-fight-backend/internal/access"
	"code-fight-backend/internal/judge"
	"code-fight-backend/internal/matches"
	"code-fight-backend/internal/submissions"
	"code-fight-backend/internal/testsupport"

	"github.com/google/uuid"
)

func TestScheduledTournamentWaitsForPreDeadlineVerdict(t *testing.T) {
	db, ctx := testsupport.RequirePostgres(t)
	service := NewService(db)
	organizer := createTournamentPlayer(t, db, ctx, access.Organizer)
	player := createTournamentPlayer(t, db, ctx, access.User)
	opponent := createTournamentPlayer(t, db, ctx, access.User)
	draft, err := service.CreateDraft(ctx, organizer, CreateInput{Slug: "settle-" + uuid.NewString(), Title: "Settlement cup"}, "settlement-test")
	if err != nil {
		t.Fatal(err)
	}
	var matchID uuid.UUID
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, "delete from submissions where match_id=$1", matchID)
		_, _ = db.Exec(ctx, "delete from tournaments where id=$1", draft.ID)
		_, _ = db.Exec(ctx, "delete from matches where id=$1", matchID)
	})
	now := time.Now().UTC()
	closes, starts, ends := now.Add(time.Minute), now.Add(2*time.Minute), now.Add(5*time.Minute)
	if err := service.Transition(ctx, organizer, draft.ID, TransitionInput{
		Status: "published", TaskSlug: "signal-stabilizer", ExpectedConfigVersion: 1,
		Schedule: Schedule{RegistrationClosesAt: &closes, StartsAt: &starts, EndsAt: &ends},
	}, "settlement-test"); err != nil {
		t.Fatal(err)
	}
	for _, participant := range []uuid.UUID{player, opponent} {
		if err := service.Join(ctx, participant, draft.ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(ctx, `update tournaments set registration_closes_at=now()-interval '20 seconds',
		starts_at=now()-interval '10 seconds' where id=$1`, draft.ID); err != nil {
		t.Fatal(err)
	}
	if advanced, err := service.advanceSchedule(ctx); !advanced || err != nil {
		t.Fatalf("start tournament: %v, %v", advanced, err)
	}
	board, err := service.Board(ctx, draft.ID)
	if err != nil || len(board.Fixtures) != 1 {
		t.Fatalf("scheduled board: %#v, %v", board, err)
	}
	matchID, err = service.ScheduleFixture(ctx, organizer, draft.ID, board.Fixtures[0].ID, "settlement-test")
	if err != nil {
		t.Fatal(err)
	}
	matchRepo := matches.NewRepository(db)
	for _, participant := range []uuid.UUID{player, opponent} {
		if _, err := matchRepo.StartMatch(ctx, matchID, participant, time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	subRepo := submissions.NewPGRepository(db)
	matchIDString := matchID.String()
	accepted := submissions.Submission{
		ID: uuid.NewString(), MatchID: &matchIDString, TaskID: board.Tournament.TaskID.String(),
		UserID: player.String(), Status: submissions.StatusQueued, SourceCode: "package main",
		Language: "go", LanguageVersion: "1.24", CreatedAt: now, UpdatedAt: now,
	}
	if err := subRepo.Create(ctx, &accepted); err != nil {
		t.Fatal(err)
	}
	var realDeadline time.Time
	if err := db.QueryRow(ctx, `update tournaments set ends_at=clock_timestamp()-interval '1 second'
		where id=$1 returning ends_at`, draft.ID).Scan(&realDeadline); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, "update submissions set created_at=$2 where id=$1", accepted.ID, realDeadline.Add(-time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if advanced, err := service.advanceSchedule(ctx); advanced || err != nil {
		t.Fatalf("unfinished eligible submission must defer finalization: %v, %v", advanced, err)
	}
	late := accepted
	late.ID = uuid.NewString()
	late.CreatedAt = time.Now().UTC()
	if err := subRepo.Create(ctx, &late); !errors.Is(err, submissions.ErrMatchSubmissionDeadlinePassed) {
		t.Fatalf("late submission admitted: %v", err)
	}
	judgeID := uuid.NewString()
	verdict := submissions.VerdictAccepted
	started := realDeadline.Add(time.Millisecond)
	finished := time.Now().UTC()
	matchService := matches.NewService(matchRepo, nil, submissions.NewMatchReader(subRepo))
	callbacks := submissions.NewService(subRepo, nil, nil, nil, matchService, "", "", 0, nil)
	if err := callbacks.HandleJudgeCallback(ctx, judge.CallbackPayload{
		SubmissionID: accepted.ID, JudgeSubmissionID: judgeID, JudgeStatus: submissions.JudgeStatusFinished,
		Verdict: &verdict, StartedAt: &started, FinishedAt: &finished,
	}); err != nil {
		t.Fatalf("apply the delayed Judge verdict: %v", err)
	}
	if advanced, err := service.advanceSchedule(ctx); !advanced || err != nil {
		t.Fatalf("complete after verdict: %v, %v", advanced, err)
	}
	board, err = service.Board(ctx, draft.ID)
	if err != nil || board.Tournament.Status != "completed" || len(board.Standings) != 2 ||
		board.Standings[0].UserID != player || board.Standings[0].Points != 3 ||
		board.Standings[0].ContestRatingAfter == nil || *board.Standings[0].ContestRatingAfter != 1016 {
		t.Fatalf("late verdict missing from the final standings: %#v, %v", board, err)
	}
}

func TestScheduledTournamentRegistrationAndExpiry(t *testing.T) {
	db, ctx := testsupport.RequirePostgres(t)
	service := NewService(db)
	organizer := createTournamentPlayer(t, db, ctx, access.Organizer)
	player := createTournamentPlayer(t, db, ctx, access.User)
	opponent := createTournamentPlayer(t, db, ctx, access.User)
	late := createTournamentPlayer(t, db, ctx, access.User)
	draft, err := service.CreateDraft(ctx, organizer, CreateInput{Slug: "schedule-" + uuid.NewString(), Title: "Scheduled cup"}, "schedule-test")
	if err != nil {
		t.Fatal(err)
	}
	id := draft.ID
	var matchID uuid.UUID
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, "delete from tournaments where id=$1", id)
		_, _ = db.Exec(ctx, "delete from matches where id=$1", matchID)
	})
	now := time.Now().In(time.FixedZone("test offset", 2*60*60)).Truncate(time.Microsecond)
	closes, starts, ends := now.Add(time.Minute), now.Add(2*time.Minute), now.Add(5*time.Minute)
	published := TransitionInput{Status: "published", TaskSlug: "signal-stabilizer", ExpectedConfigVersion: 1}
	published.Schedule = Schedule{RegistrationClosesAt: &closes, StartsAt: &starts}
	if err := service.Transition(ctx, organizer, id, published, "schedule-test"); !errors.Is(err, access.ErrInvalidInput) {
		t.Fatalf("incomplete schedule = %v", err)
	}
	published.EndsAt = &ends
	if err := service.Transition(ctx, organizer, id, published, "schedule-test"); err != nil {
		t.Fatal(err)
	}
	board, err := service.Board(ctx, id)
	if err != nil || board.Tournament.StartsAt == nil || !board.Tournament.StartsAt.Equal(starts) ||
		board.Tournament.StartsAt.Location() != time.UTC {
		t.Fatalf("published schedule = %#v, %v", board.Tournament, err)
	}
	joinErrors := make(chan error, 3)
	for _, participant := range []uuid.UUID{player, opponent, player} {
		go func() { joinErrors <- service.Join(ctx, participant, id) }()
	}
	for range 3 {
		if err := <-joinErrors; err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(ctx, "update tournaments set registration_closes_at=now()-interval '20 seconds' where id=$1", id); err != nil {
		t.Fatal(err)
	}
	if err := service.Join(ctx, late, id); !errors.Is(err, access.ErrConflict) {
		t.Fatalf("late registration = %v", err)
	}
	if advanced, err := service.advanceSchedule(ctx); advanced || err != nil {
		t.Fatalf("advanced before start = %t, %v", advanced, err)
	}
	if _, err := db.Exec(ctx, "update tournaments set starts_at=now()-interval '10 seconds' where id=$1", id); err != nil {
		t.Fatal(err)
	}

	type advanceResult struct {
		moved bool
		err   error
	}
	calls := make(chan advanceResult, 2)
	for range 2 {
		go func() {
			moved, err := service.advanceSchedule(ctx)
			calls <- advanceResult{moved, err}
		}()
	}
	transitions := 0
	for range 2 {
		result := <-calls
		if result.err != nil {
			t.Fatal(result.err)
		}
		if result.moved {
			transitions++
		}
	}
	board, err = service.Board(ctx, id)
	if err != nil || transitions != 1 || board.Tournament.Status != "running" ||
		len(board.Fixtures) != 1 || len(board.Standings) != 2 {
		t.Fatalf("automatic start = %#v, transitions=%d, %v", board, transitions, err)
	}
	matchID, err = service.ScheduleFixture(ctx, organizer, id, board.Fixtures[0].ID, "schedule-test")
	if err != nil {
		t.Fatal(err)
	}
	board, err = service.Board(ctx, id)
	if err != nil || len(board.Fixtures) != 1 || board.Fixtures[0].ReadyDeadlineAt == nil ||
		!board.Fixtures[0].ReadyDeadlineAt.Equal(ends) {
		t.Fatalf("readiness deadline = %#v, %v", board.Fixtures, err)
	}
	if _, err := db.Exec(ctx, "update tournaments set ends_at=now()-interval '1 second' where id=$1", id); err != nil {
		t.Fatal(err)
	}
	matchRepo := matches.NewRepository(db)
	if _, err := matchRepo.StartMatch(ctx, matchID, player, time.Minute); !errors.Is(err, matches.ErrMatchCancelled) {
		t.Fatalf("accepted readiness after its deadline: %v", err)
	}
	if advanced, err := service.advanceSchedule(ctx); !advanced || err != nil {
		t.Fatalf("advance expired tournament = %t, %v", advanced, err)
	}
	board, err = service.Board(ctx, id)
	if err != nil || board.Tournament.Status != "completed" || board.Tournament.CompletedAt == nil ||
		len(board.Fixtures) != 1 || board.Fixtures[0].Status != "cancelled" {
		t.Fatalf("expired tournament = %#v, %v", board, err)
	}
	page, err := service.History(ctx, player, 1, "")
	if err != nil || len(page.Items) != 1 || page.Items[0].TournamentID != id || page.Items[0].JoinedAt == nil ||
		page.Items[0].CompletedAt == nil || page.Items[0].Played != 0 {
		t.Fatalf("participation history = %#v, %v", page, err)
	}
	page, err = service.History(ctx, late, 1, "")
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("non-participant history = %#v, %v", page, err)
	}
}
