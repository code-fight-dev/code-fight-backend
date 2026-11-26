package tournament

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"code-fight-backend/internal/access"
	"code-fight-backend/internal/audit"
	"code-fight-backend/internal/managementtx"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Schedule struct {
	RegistrationClosesAt *time.Time `json:"registrationClosesAt"`
	StartsAt             *time.Time `json:"startsAt"`
	EndsAt               *time.Time `json:"endsAt"`
}

func (s Schedule) empty() bool {
	return s.RegistrationClosesAt == nil && s.StartsAt == nil && s.EndsAt == nil
}

func (s Schedule) validate() error {
	if s.empty() {
		return nil
	}
	if s.RegistrationClosesAt == nil || s.StartsAt == nil || s.EndsAt == nil ||
		s.RegistrationClosesAt.After(*s.StartsAt) || !s.StartsAt.Before(*s.EndsAt) {
		return access.ErrInvalidInput
	}
	return nil
}

func (s Schedule) utc() Schedule {
	return Schedule{utcTime(s.RegistrationClosesAt), utcTime(s.StartsAt), utcTime(s.EndsAt)}
}

func utcTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}

func createFixtures(ctx context.Context, tx pgx.Tx, id uuid.UUID) (bool, error) {
	result, err := tx.Exec(ctx, `insert into tournament_fixtures (tournament_id,player1_id,player2_id)
		select $1,p1.user_id,p2.user_id from tournament_participants p1
		join tournament_participants p2 on p2.tournament_id=p1.tournament_id and p1.user_id<p2.user_id
		where p1.tournament_id=$1`, id)
	return result.RowsAffected() > 0, err
}

func updateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string) error {
	_, err := tx.Exec(ctx, `update tournaments set status=$2, config_version=config_version+1,
		completed_at=case when $2='completed' then clock_timestamp() else completed_at end,
		updated_at=clock_timestamp() where id=$1`, id, status)
	return err
}

func (s *Service) RunSchedule(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for ctx.Err() == nil {
		for range 20 {
			advanced, err := s.advanceSchedule(ctx)
			if err != nil && ctx.Err() == nil {
				logger.Warn("scheduled tournament transition failed", "error", err)
			}
			if !advanced || err != nil {
				break
			}
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) advanceSchedule(ctx context.Context) (bool, error) {
	ctx, tx, cleanup, err := managementtx.Begin(ctx, s.db)
	if err != nil {
		return false, err
	}
	defer cleanup()

	var id uuid.UUID
	var previous string
	var deadlinePassed bool
	err = tx.QueryRow(ctx, `select t.id,t.status,t.ends_at <= clock_timestamp()
		from tournaments t
		where t.starts_at is not null and (
			(t.status='published' and t.starts_at <= clock_timestamp())
			or (t.status='running' and (
				(t.ends_at <= clock_timestamp() and not `+tournamentHasPendingSubmissions+`)
				or not exists (
					select 1 from tournament_fixtures f left join matches m on m.id=f.match_id
					where f.tournament_id=t.id and m.status is distinct from 'finished'
				)
			))
		) order by t.starts_at,t.id limit 1 for update of t skip locked`).Scan(&id, &previous, &deadlinePassed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	next := "completed"
	reason := "all_fixtures_finished"
	if deadlinePassed {
		reason = "deadline_passed"
		// Match submission admission takes the same row lock. Recheck after
		// waiting for any submission that was committing as the deadline passed.
		if _, err := tx.Exec(ctx, `select m.id from tournament_fixtures f join matches m on m.id=f.match_id
			where f.tournament_id=$1 and m.status in ('pending','running') order by m.id for update of m`, id); err != nil {
			return false, err
		}
		var pending bool
		if err := tx.QueryRow(ctx, `select `+tournamentHasPendingSubmissions+` from tournaments t where t.id=$1`, id).Scan(&pending); err != nil {
			return false, err
		}
		if pending {
			return false, nil
		}
		if _, err := tx.Exec(ctx, `update matches m set status='cancelled', result_type='cancelled',
			winning_reason='cancelled', judge_status='failed', finished_at=clock_timestamp(), updated_at=clock_timestamp()
			from tournament_fixtures f where f.match_id=m.id and f.tournament_id=$1
			and m.status in ('pending','running')`, id); err != nil {
			return false, err
		}
	} else if previous == "published" {
		created, err := createFixtures(ctx, tx, id)
		if err != nil {
			return false, err
		}
		reason = "insufficient_participants"
		if created {
			next = "running"
			reason = "scheduled_start"
		}
	}
	if err := updateStatus(ctx, tx, id, next); err != nil {
		return false, err
	}
	if err := audit.Append(ctx, tx, audit.Event{
		Action: "tournament." + next, EntityType: "tournament", EntityID: id,
		TournamentID: &id, RequestID: uuid.NewString(),
		Metadata: map[string]any{"previousStatus": previous, "status": next, "reason": reason},
	}); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

const tournamentHasPendingSubmissions = `exists (
	select 1 from tournament_fixtures f join matches m on m.id=f.match_id
	where f.tournament_id=t.id and m.status='running'
	and tournament_match_waits_for_judge(m.id, least(m.expires_at, t.ends_at))
)`
