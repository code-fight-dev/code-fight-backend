package tournament

import (
	"context"
	"strings"

	"code-fight-backend/internal/access"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const sharedProblemSet = "shared_problem_set"
const maxContestProblems = 20

type ContestProblem struct {
	TaskID     uuid.UUID `json:"taskId"`
	Slug       string    `json:"slug"`
	Title      string    `json:"title"`
	Difficulty string    `json:"difficulty"`
	Position   int       `json:"position"`
}

func publishSharedProblemSet(ctx context.Context, tx pgx.Tx, id uuid.UUID, schedule Schedule, input TransitionInput) error {
	if schedule.empty() || input.TaskSlug != "" || len(input.TaskSlugs) == 0 || len(input.TaskSlugs) > maxContestProblems {
		return access.ErrInvalidInput
	}
	slugs := make([]string, 0, len(input.TaskSlugs))
	seen := make(map[string]bool, len(input.TaskSlugs))
	for _, value := range input.TaskSlugs {
		slug := strings.TrimSpace(value)
		if !validSlug(slug) || seen[slug] {
			return access.ErrInvalidInput
		}
		seen[slug] = true
		slugs = append(slugs, slug)
	}
	result, err := tx.Exec(ctx, `update tournaments set format=$2, status='published',
		registration_closes_at=$3, starts_at=$4, ends_at=$5
		where id=$1 and $3::timestamptz > clock_timestamp()`,
		id, sharedProblemSet, schedule.RegistrationClosesAt, schedule.StartsAt, schedule.EndsAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return access.ErrInvalidInput
	}
	result, err = tx.Exec(ctx, `insert into tournament_problems (tournament_id,task_id,position)
		select $1, t.id, p.position from unnest($2::text[]) with ordinality p(slug,position)
		join tasks t on t.slug=p.slug and t.is_active
		where exists (select 1 from task_languages l where l.task_id=t.id and l.is_enabled)`, id, slugs)
	if err != nil {
		return err
	}
	if result.RowsAffected() != int64(len(slugs)) {
		return access.ErrInvalidInput
	}
	return nil
}

func sharedProblemBoard(ctx context.Context, tx pgx.Tx, summary Summary) (Board, error) {
	rows, err := tx.Query(ctx, `select p.task_id, t.slug, t.title, t.difficulty, p.position
		from tournament_problems p join tasks t on t.id=p.task_id
		join tournaments c on c.id=p.tournament_id
		where c.id=$1 and c.starts_at <= transaction_timestamp() order by p.position`, summary.ID)
	if err != nil {
		return Board{}, err
	}
	problems, err := pgx.CollectRows(rows, pgx.RowToStructByName[ContestProblem])
	if err != nil {
		return Board{}, err
	}
	standings := sharedProblemStandingsSQL
	if summary.Status == "completed" {
		standings = `with standings as (select user_id, rank, solved, penalty_seconds, judged_submissions
			from tournament_problem_results where tournament_id=$1)`
	}
	rows, err = tx.Query(ctx, standings+`
		select s.user_id, u.username, s.rank, s.solved, s.penalty_seconds, s.judged_submissions, s.solved as points,
			c.rating_before as contest_rating_before, c.rating_after as contest_rating_after
		from standings s join users u on u.id=s.user_id
		left join contest_rating_changes c on c.tournament_id=$1 and c.user_id=s.user_id
		order by s.rank, u.username, s.user_id`, summary.ID)
	if err != nil {
		return Board{}, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[Standing])
	if err != nil {
		return Board{}, err
	}
	return Board{Tournament: summary, Standings: items, Fixtures: []Fixture{}, Problems: &problems}, nil
}

// The earliest accepted submission is measured by admission time. Later
// submissions cannot change that problem's score, regardless of callback order.
const sharedProblemStandingsSQL = `with judged as materialized (
	select id, user_id, task_id, created_at, verdict,
		count(*) filter (where verdict not in ('accepted','compile_error')) over (
			partition by user_id, task_id order by created_at, id rows unbounded preceding
		) as wrong_attempts
	from submissions
	where tournament_id=$1 and status='finished' and verdict <> 'system_error'
), solves as (
	select distinct on (user_id, task_id) user_id, task_id, created_at, wrong_attempts
	from judged where verdict='accepted' order by user_id, task_id, created_at, id
), scores as (
	select a.user_id, count(*) as solved, sum(
		floor(extract(epoch from a.created_at-t.starts_at))::bigint + 1200 * a.wrong_attempts
	) as penalty_seconds
	from solves a join tournaments t on t.id=$1 group by a.user_id
), attempts as (
	select user_id, count(*) as judged_submissions from judged group by user_id
), standings as (
	select p.user_id, coalesce(s.solved,0) as solved, coalesce(s.penalty_seconds,0)::bigint as penalty_seconds,
		coalesce(a.judged_submissions,0) as judged_submissions,
		dense_rank() over (order by coalesce(s.solved,0) desc, coalesce(s.penalty_seconds,0)) as rank
	from tournament_participants p left join scores s on s.user_id=p.user_id
	left join attempts a on a.user_id=p.user_id where p.tournament_id=$1
)`

func snapshotProblemStandings(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, sharedProblemStandingsSQL+`
		insert into tournament_problem_results (tournament_id,user_id,rank,solved,penalty_seconds,judged_submissions)
		select $1,user_id,rank,solved,penalty_seconds,judged_submissions from standings`, id)
	return err
}
