package tournament

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	"code-fight-backend/internal/managementtx"

	"github.com/google/uuid"
)

type Participation struct {
	Schedule
	TournamentID        uuid.UUID  `json:"tournamentId"`
	Slug                string     `json:"slug"`
	Title               string     `json:"title"`
	Status              string     `json:"status"`
	Format              string     `json:"format"`
	JoinedAt            *time.Time `json:"joinedAt"`
	CompletedAt         *time.Time `json:"completedAt"`
	Played              int        `json:"played"`
	Wins                int        `json:"wins"`
	Draws               int        `json:"draws"`
	Points              int        `json:"points"`
	ContestRatingBefore *int       `json:"contestRatingBefore"`
	ContestRatingAfter  *int       `json:"contestRatingAfter"`
	Rank                *int       `json:"rank,omitempty"`
	Solved              *int       `json:"solved,omitempty"`
	PenaltySeconds      *int64     `json:"penaltySeconds,omitempty"`
}

type HistoryPage struct {
	Items      []Participation `json:"items"`
	NextCursor string          `json:"nextCursor,omitempty"`
}

func (s *Service) History(ctx context.Context, userID uuid.UUID, limit int, rawCursor string) (HistoryPage, error) {
	after, err := decodeCursor(limit, rawCursor)
	if err != nil {
		return HistoryPage{}, err
	}
	var before *time.Time
	if !after.CreatedAt.IsZero() {
		before = &after.CreatedAt
	}
	ctx, tx, cleanup, err := managementtx.Begin(ctx, s.db)
	if err != nil {
		return HistoryPage{}, err
	}
	defer cleanup()
	rows, err := tx.Query(ctx, `select t.id,t.slug,t.title,t.status,p.joined_at,t.completed_at,
		t.registration_closes_at,t.starts_at,t.ends_at,score.played,score.wins,score.draws,coalesce(p.joined_at,t.created_at),
		c.rating_before,c.rating_after,t.format,r.rank,r.solved,r.penalty_seconds
		from tournament_participants p join tournaments t on t.id=p.tournament_id
		left join contest_rating_changes c on c.tournament_id=t.id and c.user_id=p.user_id
		left join tournament_problem_results r on r.tournament_id=t.id and r.user_id=p.user_id
		cross join lateral (
			select count(*) as played, count(*) filter (where m.winner_id=p.user_id) as wins,
				count(*) filter (where m.result_type='draw') as draws
			from tournament_fixtures f join matches m on m.id=f.match_id and m.status='finished'
			where f.tournament_id=t.id and p.user_id in (f.player1_id,f.player2_id)
		) score
		where p.user_id=$1 and t.status <> 'draft'
		  and ($2::timestamptz is null or (coalesce(p.joined_at,t.created_at),t.id)<($2,$3))
		order by coalesce(p.joined_at,t.created_at) desc,t.id desc limit $4`, userID, before, after.ID, limit+1)
	if err != nil {
		return HistoryPage{}, err
	}
	defer rows.Close()
	page := HistoryPage{Items: make([]Participation, 0)}
	var last cursor
	for rows.Next() {
		var item Participation
		var orderedAt time.Time
		if err := rows.Scan(&item.TournamentID, &item.Slug, &item.Title, &item.Status, &item.JoinedAt, &item.CompletedAt,
			&item.RegistrationClosesAt, &item.StartsAt, &item.EndsAt, &item.Played, &item.Wins, &item.Draws, &orderedAt,
			&item.ContestRatingBefore, &item.ContestRatingAfter, &item.Format,
			&item.Rank, &item.Solved, &item.PenaltySeconds); err != nil {
			return HistoryPage{}, err
		}
		if len(page.Items) == limit {
			data, err := json.Marshal(last)
			if err != nil {
				return HistoryPage{}, err
			}
			page.NextCursor = base64.RawURLEncoding.EncodeToString(data)
			break
		}
		last = cursor{CreatedAt: orderedAt, ID: item.TournamentID}
		item.JoinedAt = utcTime(item.JoinedAt)
		item.CompletedAt = utcTime(item.CompletedAt)
		item.Schedule = item.Schedule.utc()
		item.Points = 3*item.Wins + item.Draws
		if item.Solved != nil {
			item.Points = *item.Solved
		}
		page.Items = append(page.Items, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return HistoryPage{}, err
	}
	return page, tx.Commit(ctx)
}
