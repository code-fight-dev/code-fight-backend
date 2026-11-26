package tournament

import (
	"testing"
	"time"

	"code-fight-backend/internal/access"
	"code-fight-backend/internal/matches"
	"code-fight-backend/internal/testsupport"
	"github.com/google/uuid"
)

func TestBracketSeedPlacement(t *testing.T) {
	for n := 2; n <= maxParticipants; n++ {
		seeds := seedOrder(n)
		seen := map[int]bool{}
		for i, seed := range seeds {
			if seen[seed] || seed < 1 || seed > len(seeds) {
				t.Fatalf("invalid seeds for %d: %v", n, seeds)
			}
			seen[seed] = true
			if i%2 == 1 && seed > n && seeds[i-1] > n {
				t.Fatalf("empty first-round pair for %d: %v", n, seeds)
			}
		}
	}
}

func TestBracketLifecycle(t *testing.T) {
	for _, scenario := range []string{"final", "deadline"} {
		t.Run(scenario, func(t *testing.T) {
			db, ctx := testsupport.RequirePostgres(t)
			s := NewService(db)
			owner := createTournamentPlayer(t, db, ctx, access.Organizer)
			d, err := s.CreateDraft(ctx, owner, CreateInput{Slug: "bracket-" + uuid.NewString(), Title: "Cup"}, "test")
			if err != nil {
				t.Fatal(err)
			}
			var matchIDs []uuid.UUID
			defer func() {
				if _, err := db.Exec(ctx, "delete from tournaments where id=$1", d.ID); err != nil {
					t.Error(err)
				}
				for _, mid := range matchIDs {
					_, _ = db.Exec(ctx, "delete from matches where id=$1", mid)
				}
			}()
			start := time.Now().Add(time.Hour)
			end := start.Add(time.Hour)
			if err := s.Transition(ctx, owner, d.ID, TransitionInput{Status: "published", Format: singleElimination, TaskSlug: "mirror-log", ExpectedConfigVersion: 1,
				Schedule: Schedule{RegistrationClosesAt: &start, StartsAt: &start, EndsAt: &end}}, "test"); err != nil {
				t.Fatal(err)
			}
			for range 3 {
				player := createTournamentPlayer(t, db, ctx, access.User)
				if err := s.Join(ctx, player, d.ID); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := db.Exec(ctx, `update tournaments set starts_at=now()-interval '1 hour',registration_closes_at=now()-interval '1 hour' where id=$1`, d.ID); err != nil {
				t.Fatal(err)
			}
			step := func() {
				t.Helper()
				if advanced, err := s.advanceSchedule(ctx); !advanced || err != nil {
					t.Fatalf("advance = %v, %v", advanced, err)
				}
			}
			board := func() Board {
				t.Helper()
				b, err := s.Board(ctx, d.ID)
				if err != nil || b.Bracket == nil || len(b.Bracket.Slots) != 3 {
					t.Fatalf("board = %+v, %v", b, err)
				}
				return b
			}
			step()
			b := board()
			var semi BracketSlot
			for _, slot := range b.Bracket.Slots {
				if slot.Status == "ready" {
					semi = slot
				}
			}
			if semi.FixtureID == nil || semi.NextSlotID == nil || *semi.NextSlotID != b.Bracket.Slots[2].ID || len(b.Fixtures) != 1 {
				t.Fatal("incorrect initial tree", b.Bracket)
			}
			play := func(fixture uuid.UUID, draw bool) uuid.UUID {
				t.Helper()
				mid, err := s.ScheduleFixture(ctx, owner, d.ID, fixture, "test")
				if err != nil {
					t.Fatal(err)
				}
				matchIDs = append(matchIDs, mid)
				repo := matches.NewRepository(db)
				m, err := repo.GetMatch(ctx, mid)
				if err != nil {
					t.Fatal(err)
				}
				outcome := matches.FinishMatchInput{MatchID: mid, WinnerID: &m.Player1ID, ResultType: matches.MatchResultTypePlayer1Win, WinningReason: matches.WinningReasonSurrender}
				if draw {
					outcome.WinnerID = nil
					outcome.ResultType = matches.MatchResultTypeDraw
					outcome.WinningReason = matches.WinningReasonDraw
				}
				if err := repo.FinishMatch(ctx, outcome); err != nil {
					t.Fatal(err)
				}
				return m.Player1ID
			}
			play(*semi.FixtureID, true)
			if advanced, err := s.advanceSchedule(ctx); advanced || err != nil {
				t.Fatal("draw advanced", advanced, err)
			}
			play(*semi.FixtureID, false)
			step()
			final := board().Bracket.Slots[2]
			if final.FixtureID == nil || final.Player1ID == nil || final.Player2ID == nil {
				t.Fatal("missing finalists", final)
			}
			if scenario == "deadline" {
				if _, err := db.Exec(ctx, "update tournaments set ends_at=now() where id=$1", d.ID); err != nil {
					t.Fatal(err)
				}
				step()
				b = board()
				if b.Tournament.Status != "completed" || b.Bracket.ChampionID != nil || b.Standings[0].ContestRatingAfter != nil {
					t.Fatal("unfinished final awarded", b)
				}
				return
			}
			winner := play(*final.FixtureID, false)
			step()
			b = board()
			if b.Tournament.Status != "completed" || b.Bracket.ChampionID == nil || *b.Bracket.ChampionID != winner || b.Standings[0].UserID != winner || b.Standings[0].ContestRatingAfter == nil {
				t.Fatal("incorrect final result", b)
			}
		})
	}
}
