package tournament

import (
	"context"
	"errors"
	"testing"
	"time"

	"code-fight-backend/internal/access"
	"code-fight-backend/internal/auth"
	"code-fight-backend/internal/matches"
	"code-fight-backend/internal/testsupport"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTournamentRoundRobinLifecycle(t *testing.T) {
	db, ctx := testsupport.RequirePostgres(t)
	service := NewService(db)
	organizer := createTournamentPlayer(t, db, ctx, access.Organizer)
	otherOrganizer := createTournamentPlayer(t, db, ctx, access.Organizer)
	player1 := createTournamentPlayer(t, db, ctx, access.User)
	player2 := createTournamentPlayer(t, db, ctx, access.User)
	draft, err := service.CreateDraft(ctx, organizer, CreateInput{Slug: "cup-" + uuid.NewString(), Title: "Local cup"}, "round-robin-test")
	if err != nil {
		t.Fatal(err)
	}
	var matchID uuid.UUID
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, "delete from tournaments where id=$1", draft.ID)
		_, _ = db.Exec(ctx, "delete from matches where id=$1", matchID)
	})
	if _, err := service.Board(ctx, draft.ID); !errors.Is(err, access.ErrNotFound) {
		t.Fatalf("draft is publicly visible: %v", err)
	}

	publish := TransitionInput{Status: "published", TaskSlug: "signal-stabilizer", ExpectedConfigVersion: 1}
	if err := service.Transition(ctx, otherOrganizer, draft.ID, publish, "round-robin-test"); !errors.Is(err, access.ErrNotFound) {
		t.Fatalf("another organizer can publish this tournament: %v", err)
	}
	if err := service.Transition(ctx, organizer, draft.ID, publish, "round-robin-test"); err != nil {
		t.Fatal(err)
	}
	start := TransitionInput{Status: "running", ExpectedConfigVersion: 2}
	if err := service.Transition(ctx, organizer, draft.ID, start, "round-robin-test"); !errors.Is(err, access.ErrConflict) {
		t.Fatalf("started without participants: %v", err)
	}
	for _, player := range []uuid.UUID{player1, player2, player1} {
		if err := service.Join(ctx, player, draft.ID); err != nil {
			t.Fatal(err)
		}
	}
	if err := service.Transition(ctx, organizer, draft.ID, start, "round-robin-test"); err != nil {
		t.Fatal(err)
	}
	if err := service.Join(ctx, organizer, draft.ID); !errors.Is(err, access.ErrConflict) {
		t.Fatalf("joined after registration closed: %v", err)
	}

	board, err := service.Board(ctx, draft.ID)
	if err != nil || len(board.Fixtures) != 1 || len(board.Standings) != 2 {
		t.Fatalf("running board: %#v, %v", board, err)
	}
	finalize := TransitionInput{Status: "completed", ExpectedConfigVersion: 3}
	if err := service.Transition(ctx, organizer, draft.ID, finalize, "round-robin-test"); !errors.Is(err, access.ErrConflict) {
		t.Fatalf("completed with unfinished fixtures: %v", err)
	}
	matchID, err = service.ScheduleFixture(ctx, organizer, draft.ID, board.Fixtures[0].ID, "round-robin-test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ScheduleFixture(ctx, organizer, draft.ID, board.Fixtures[0].ID, "round-robin-test"); !errors.Is(err, access.ErrConflict) {
		t.Fatalf("scheduled an already active pairing: %v", err)
	}
	matchRepo := matches.NewRepository(db)
	if _, err := matchRepo.StartMatch(ctx, matchID, player1, time.Minute); err != nil {
		t.Fatal(err)
	}
	match, err := matchRepo.StartMatch(ctx, matchID, player2, time.Minute)
	if err != nil || match.IsRated {
		t.Fatalf("tournament match: %#v, %v", match, err)
	}
	if err := matchRepo.FinishMatch(ctx, matches.FinishMatchInput{
		MatchID: matchID, WinnerID: &match.Player1ID,
		ResultType: matches.MatchResultTypePlayer1Win, WinningReason: matches.WinningReasonAcceptedFaster,
	}); err != nil {
		t.Fatal(err)
	}
	if err := service.Transition(ctx, organizer, draft.ID, finalize, "round-robin-test"); err != nil {
		t.Fatal(err)
	}
	board, err = service.Board(ctx, draft.ID)
	if err != nil || board.Tournament.Status != "completed" ||
		len(board.Standings) != 2 || board.Standings[0].UserID != match.Player1ID ||
		board.Standings[0].Points != 3 || board.Standings[1].Points != 0 {
		t.Fatalf("final board: %#v, %v", board, err)
	}
	var history int
	if err := db.QueryRow(ctx, "select count(*) from rating_history where match_id=$1", matchID).Scan(&history); err != nil || history != 0 {
		t.Fatalf("tournament changed global rating: %d, %v", history, err)
	}
}

func createTournamentPlayer(t *testing.T, db *pgxpool.Pool, ctx context.Context, role access.Role) uuid.UUID {
	t.Helper()
	id := uuid.New()
	name := "cup_" + id.String()[:8]
	_, err := auth.NewRepository(db).CreateUser(ctx, auth.User{
		ID: id, Username: name, Email: name + "@example.com", DisplayName: name,
		PasswordHash: "test", AvatarSource: auth.AvatarSourceNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec(ctx, "delete from users where id=$1", id) })
	if _, err := db.Exec(ctx, "update users set role=$2 where id=$1", id, role); err != nil {
		t.Fatal(err)
	}
	return id
}
