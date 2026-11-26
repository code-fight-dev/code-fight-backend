package tournament

import (
	"context"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const initialContestRating = 1000
const contestRatingFactor = 32

type ratedParticipant struct {
	UserID uuid.UUID
	Rank   int
	Rating int
}

func contestRatingChanges(field []ratedParticipant) []int {
	changes := make([]int, len(field))
	if len(field) < 2 {
		return changes
	}
	for i, player := range field {
		var difference float64
		for j, opponent := range field {
			if i == j {
				continue
			}
			actual := 0.5
			switch {
			case player.Rank < opponent.Rank:
				actual = 1
			case player.Rank > opponent.Rank:
				actual = 0
			}
			expected := 1 / (1 + math.Pow(10, (float64(opponent.Rating)-float64(player.Rating))/400))
			difference += actual - expected
		}
		changes[i] = int(math.Round(contestRatingFactor * difference / float64(len(field)-1)))
	}
	return changes
}

func applyContestRating(ctx context.Context, tx pgx.Tx, tournamentID uuid.UUID) error {
	// The completion transaction holds the tournament lock. Serialize rating
	// snapshots across events so concurrent tournaments cannot lose updates.
	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended('contest-rating', 0))`); err != nil {
		return err
	}
	claimed, err := tx.Exec(ctx, `insert into contest_rating_events (tournament_id)
		values ($1) on conflict do nothing`, tournamentID)
	if err != nil || claimed.RowsAffected() == 0 {
		return err
	}
	rows, err := tx.Query(ctx, contestStandingsSQL+bracketRanksSQL+`
		select s.user_id, coalesce(b.rank,dense_rank() over (order by s.points desc, s.wins desc)) as rank,
			coalesce(r.rating,$2) as rating
		from standings s left join contest_ratings r on r.user_id=s.user_id
		join tournaments t on t.id=$1 left join bracket_ranks b on b.user_id=s.user_id
		where s.played > 0 and (t.format='round_robin' or (b.has_champion and b.has_decisive_match))
		union all
		select s.user_id, s.rank, coalesce(r.rating,$2) as rating
		from tournament_problem_results s left join contest_ratings r on r.user_id=s.user_id
		where s.tournament_id=$1 and s.judged_submissions > 0
		order by user_id`, tournamentID, initialContestRating)
	if err != nil {
		return err
	}
	field, err := pgx.CollectRows(rows, pgx.RowToStructByName[ratedParticipant])
	if err != nil || len(field) < 2 {
		return err
	}
	changes := contestRatingChanges(field)
	for i, player := range field {
		after := player.Rating + changes[i]
		if _, err := tx.Exec(ctx, `insert into contest_ratings (user_id,rating) values ($1,$2)
			on conflict (user_id) do update set rating=excluded.rating`, player.UserID, after); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `insert into contest_rating_changes (tournament_id,user_id,rating_before,rating_after)
			values ($1,$2,$3,$4)`, tournamentID, player.UserID, player.Rating, after); err != nil {
			return err
		}
	}
	return nil
}
