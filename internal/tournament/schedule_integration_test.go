package tournament

import (
	"errors"
	"testing"
	"time"

	"code-fight-backend/internal/access"
	"code-fight-backend/internal/matches"
	"code-fight-backend/internal/testsupport"

	"github.com/google/uuid"
)

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
