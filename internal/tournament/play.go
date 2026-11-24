package tournament

import (
	"context"
	"errors"

	"code-fight-backend/internal/access"
	"code-fight-backend/internal/audit"
	"code-fight-backend/internal/managementtx"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const maxParticipants = 16

type Summary struct {
	ID            uuid.UUID `json:"id"`
	Slug          string    `json:"slug"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Status        string    `json:"status"`
	TaskID        uuid.UUID `json:"taskId"`
	OwnerID       uuid.UUID `json:"ownerId"`
	ConfigVersion int64     `json:"configVersion"`
}

type Standing struct {
	UserID   uuid.UUID `json:"userId"`
	Username string    `json:"username"`
	Played   int       `json:"played"`
	Wins     int       `json:"wins"`
	Draws    int       `json:"draws"`
	Points   int       `json:"points"`
}

type Fixture struct {
	ID       uuid.UUID  `json:"id"`
	Player1  string     `json:"player1"`
	Player2  string     `json:"player2"`
	MatchID  *uuid.UUID `json:"matchId"`
	Status   string     `json:"status"`
	WinnerID *uuid.UUID `json:"winnerId"`
}

type Board struct {
	Tournament Summary    `json:"tournament"`
	Standings  []Standing `json:"standings"`
	Fixtures   []Fixture  `json:"fixtures"`
}

type TransitionInput struct {
	Status                string `json:"status"`
	TaskSlug              string `json:"taskSlug"`
	ExpectedConfigVersion int64  `json:"expectedConfigVersion"`
}

const summaryColumns = `id, slug, title, description, status, task_id, owner_id, config_version`

func scanSummary(row interface{ Scan(...any) error }) (Summary, error) {
	var summary Summary
	err := row.Scan(&summary.ID, &summary.Slug, &summary.Title, &summary.Description,
		&summary.Status, &summary.TaskID, &summary.OwnerID, &summary.ConfigVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		err = access.ErrNotFound
	}
	return summary, err
}

func (s *Service) ListPublished(ctx context.Context) ([]Summary, error) {
	ctx, tx, cleanup, err := managementtx.Begin(ctx, s.db)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	rows, err := tx.Query(ctx, `select `+summaryColumns+` from tournaments
		where status <> 'draft' order by created_at desc, id limit 50`)
	if err != nil {
		return nil, err
	}
	items := make([]Summary, 0)
	for rows.Next() {
		summary, err := scanSummary(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, tx.Commit(ctx)
}

func (s *Service) Board(ctx context.Context, id uuid.UUID) (Board, error) {
	ctx, tx, cleanup, err := managementtx.Begin(ctx, s.db)
	if err != nil {
		return Board{}, err
	}
	defer cleanup()
	if _, err := tx.Exec(ctx, "set transaction isolation level repeatable read read only"); err != nil {
		return Board{}, err
	}
	summary, err := scanSummary(tx.QueryRow(ctx, `select `+summaryColumns+
		` from tournaments where id=$1 and status <> 'draft'`, id))
	if err != nil {
		return Board{}, err
	}
	fixtures, err := tx.Query(ctx, `
select f.id as id, p1.username as player1, p2.username as player2,
       f.match_id as match_id, coalesce(m.status, 'unscheduled') as status, m.winner_id as winner_id
from tournament_fixtures f
join users p1 on p1.id=f.player1_id
join users p2 on p2.id=f.player2_id
left join matches m on m.id=f.match_id
where f.tournament_id=$1
order by p1.username, p2.username, f.id`, id)
	if err != nil {
		return Board{}, err
	}
	fixtureItems, err := pgx.CollectRows(fixtures, pgx.RowToStructByName[Fixture])
	if err != nil {
		return Board{}, err
	}
	standings, err := tx.Query(ctx, `
select p.user_id, u.username, count(m.id) as played,
       count(*) filter (where m.winner_id=p.user_id) as wins,
       count(*) filter (where m.result_type='draw') as draws,
       3 * count(*) filter (where m.winner_id=p.user_id) +
           count(*) filter (where m.result_type='draw') as points
from tournament_participants p
join users u on u.id=p.user_id
left join tournament_fixtures f on f.tournament_id=p.tournament_id
    and p.user_id in (f.player1_id, f.player2_id)
left join matches m on m.id=f.match_id and m.status='finished'
where p.tournament_id=$1
group by p.user_id,u.username
order by points desc, wins desc, u.username, p.user_id`, id)
	if err != nil {
		return Board{}, err
	}
	standingItems, err := pgx.CollectRows(standings, pgx.RowToStructByName[Standing])
	if err != nil {
		return Board{}, err
	}
	return Board{Tournament: summary, Standings: standingItems, Fixtures: fixtureItems}, tx.Commit(ctx)
}

func (s *Service) Join(ctx context.Context, userID, id uuid.UUID) error {
	ctx, tx, cleanup, err := managementtx.Begin(ctx, s.db)
	if err != nil {
		return err
	}
	defer cleanup()
	var status string
	err = tx.QueryRow(ctx, `select status from tournaments where id=$1 and status <> 'draft' for update`, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return access.ErrNotFound
	}
	if err != nil {
		return err
	}
	var joined bool
	var count int
	err = tx.QueryRow(ctx, `select count(*), coalesce(bool_or(user_id=$2),false)
		from tournament_participants where tournament_id=$1`, id, userID).Scan(&count, &joined)
	if err != nil {
		return err
	}
	if joined {
		return tx.Commit(ctx)
	}
	if status != "published" || count >= maxParticipants {
		return access.ErrConflict
	}
	if _, err := tx.Exec(ctx, "insert into tournament_participants values ($1,$2)", id, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func lockManagedTournament(ctx context.Context, tx pgx.Tx, actorID, id uuid.UUID) (Draft, error) {
	actor, err := access.LockActor(ctx, tx, actorID)
	if err != nil {
		return Draft{}, err
	}
	if !access.CanCreateTournament(actor.Role) {
		return Draft{}, access.ErrForbidden
	}
	return scanDraft(tx.QueryRow(ctx, `select `+columns+` from tournaments
		where id=$1 and ($2::boolean or owner_id=$3) for update`, id, actor.Role == access.Admin, actor.ID))
}

func (s *Service) Transition(ctx context.Context, actorID, id uuid.UUID, input TransitionInput, requestID string) error {
	ctx, tx, cleanup, err := managementtx.Begin(ctx, s.db)
	if err != nil {
		return err
	}
	defer cleanup()
	draft, err := lockManagedTournament(ctx, tx, actorID, id)
	if err != nil {
		return err
	}
	if draft.ConfigVersion != input.ExpectedConfigVersion {
		return access.ErrConflict
	}
	switch input.Status {
	case "published":
		if draft.Status != "draft" {
			return access.ErrConflict
		}
		result, err := tx.Exec(ctx, `update tournaments set task_id=tasks.id, status='published'
			from tasks where tournaments.id=$1 and tasks.slug=$2 and tasks.is_active
			and exists (select 1 from task_languages where task_id=tasks.id and is_enabled)`, id, input.TaskSlug)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return access.ErrInvalidInput
		}
	case "running":
		if draft.Status != "published" {
			return access.ErrConflict
		}
		result, err := tx.Exec(ctx, `insert into tournament_fixtures (tournament_id,player1_id,player2_id)
			select $1,p1.user_id,p2.user_id from tournament_participants p1
			join tournament_participants p2 on p2.tournament_id=p1.tournament_id and p1.user_id<p2.user_id
			where p1.tournament_id=$1`, id)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return access.ErrConflict
		}
	case "completed":
		if draft.Status != "running" {
			return access.ErrConflict
		}
		var unfinished bool
		err := tx.QueryRow(ctx, `select exists (select 1 from tournament_fixtures f
			left join matches m on m.id=f.match_id
			where f.tournament_id=$1 and m.status is distinct from 'finished')`, id).Scan(&unfinished)
		if err != nil {
			return err
		}
		if unfinished {
			return access.ErrConflict
		}
	default:
		return access.ErrInvalidInput
	}
	if _, err := tx.Exec(ctx, `update tournaments set status=$2, config_version=config_version+1,
		updated_at=clock_timestamp() where id=$1`, id, input.Status); err != nil {
		return err
	}
	if err := audit.Append(ctx, tx, audit.Event{
		ActorID: &actorID, Action: "tournament." + input.Status, EntityType: "tournament",
		EntityID: id, TournamentID: &id, RequestID: requestID,
		Metadata: map[string]any{"previousStatus": draft.Status, "status": input.Status},
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) ScheduleFixture(ctx context.Context, actorID, tournamentID, fixtureID uuid.UUID, requestID string) (uuid.UUID, error) {
	ctx, tx, cleanup, err := managementtx.Begin(ctx, s.db)
	if err != nil {
		return uuid.Nil, err
	}
	defer cleanup()
	draft, err := lockManagedTournament(ctx, tx, actorID, tournamentID)
	if err != nil {
		return uuid.Nil, err
	}
	if draft.Status != "running" {
		return uuid.Nil, access.ErrConflict
	}
	var player1, player2 uuid.UUID
	var previousStatus string
	err = tx.QueryRow(ctx, `select f.player1_id,f.player2_id,coalesce(m.status,'unscheduled')
		from tournament_fixtures f left join matches m on m.id=f.match_id
		where f.tournament_id=$1 and f.id=$2`, tournamentID, fixtureID).Scan(&player1, &player2, &previousStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, access.ErrNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}
	if previousStatus != "unscheduled" && previousStatus != "cancelled" {
		return uuid.Nil, access.ErrConflict
	}
	// Fixture participants are stored in UUID order. Lock both in that order
	// before checking for active matches across tournaments.
	for _, player := range []uuid.UUID{player1, player2} {
		if _, err := tx.Exec(ctx,
			"select pg_advisory_xact_lock(hashtextextended('tournament-player:' || $1::text, 0))", player.String(),
		); err != nil {
			return uuid.Nil, err
		}
	}
	matchID := uuid.New()
	result, err := tx.Exec(ctx, `
insert into matches (id,task_id,task_difficulty_snapshot,player1_id,player2_id,is_rated)
select $1,t.task_id,task.difficulty,$3,$4,false
from tournaments t join tasks task on task.id=t.task_id and task.is_active
where t.id=$2 and not exists (
    select 1 from matches where status in ('pending','running')
      and (player1_id in ($3,$4) or player2_id in ($3,$4))
)`, matchID, tournamentID, player1, player2)
	if err != nil {
		return uuid.Nil, err
	}
	if result.RowsAffected() != 1 {
		return uuid.Nil, access.ErrConflict
	}
	if _, err := tx.Exec(ctx, "update tournament_fixtures set match_id=$2 where id=$1", fixtureID, matchID); err != nil {
		return uuid.Nil, err
	}
	if err := audit.Append(ctx, tx, audit.Event{
		ActorID: &actorID, Action: "tournament.match_scheduled", EntityType: "tournament",
		EntityID: tournamentID, TournamentID: &tournamentID, RequestID: requestID,
		Metadata: map[string]any{"fixtureId": fixtureID, "matchId": matchID},
	}); err != nil {
		return uuid.Nil, err
	}
	return matchID, tx.Commit(ctx)
}
