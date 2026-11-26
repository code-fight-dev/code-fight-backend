package practice

const historySQL = `
select s.id as submission_id, t.slug as task_slug, t.title as task_title, t.difficulty,
       s.language, s.language_version, s.status, s.verdict, s.created_at as submitted_at
from submissions s join tasks t on t.id = s.task_id
where s.user_id = $1 and s.match_id is null and s.tournament_id is null
  and ($2 = '' or lower(s.language) = $2)
  and ($3 = '' or s.verdict = $3)
  and ($4::timestamptz is null or (s.created_at, s.id) < ($4, $5))
order by s.created_at desc, s.id desc limit $6`

const summarySQL = `
with practice as materialized (
    select task_id, lower(language) as language, verdict, created_at
    from submissions where user_id = $1 and match_id is null and tournament_id is null
), task_progress as (
    select task_id, min(created_at) filter (where verdict = 'accepted') as first_solved_at
    from practice group by task_id
), daily as (
    select (created_at at time zone $2)::date as day, count(*) as submissions
    from practice group by 1
), first_solves as (
    select (first_solved_at at time zone $2)::date as day, count(*) as solves
    from task_progress where first_solved_at is not null group by 1
), activity as (
    select ($3::date + offset_days) as day
    from generate_series(0, $4::date - $3::date) as offset_days
), streak_islands as (
    select day, day - (row_number() over (order by day))::integer as island
    from daily where day <= $5::date
), streak_lengths as (
    select max(day) as last_day, count(*) as length from streak_islands group by island
), task_dimensions as (
    select t.id, t.is_active, d.dimension, d.key
    from tasks t cross join lateral (
        select 'difficulty', t.difficulty
        union all select 'category', nullif(btrim(t.frontend_payload->>'category'), '')
        union all
        select distinct 'tag', nullif(btrim(tag), '')
        from jsonb_array_elements_text(case when jsonb_typeof(t.frontend_payload->'tags') = 'array'
            then t.frontend_payload->'tags' else '[]'::jsonb end) as tag
    ) as d(dimension, key)
    where d.key is not null
), breakdowns as (
    select d.dimension, d.key,
        count(p.task_id) as attempted, count(p.first_solved_at) as solved,
        count(*) filter (where d.is_active) as available
    from task_dimensions d left join task_progress p on p.task_id = d.id
    group by d.dimension, d.key
    having bool_or(d.is_active) or count(p.task_id) > 0
), breakdown_json as (
    select dimension, jsonb_agg(jsonb_build_object('key', key, 'attemptedTasks', attempted,
        'solvedTasks', solved, 'availableTasks', available) order by key) as items
    from breakdowns group by dimension
), language_counts as (
    select language, count(*) as submissions, count(*) filter (where verdict = 'accepted') as accepted,
        count(distinct task_id) as attempted, count(distinct task_id) filter (where verdict = 'accepted') as solved
    from practice group by language
)
select jsonb_build_object(
    'totals', jsonb_build_object('submissions', (select count(*) from practice),
        'attemptedTasks', (select count(*) from task_progress), 'solvedTasks', (select count(first_solved_at) from task_progress)),
    'activity', jsonb_build_object('from', $3::date, 'to', $4::date, 'timeZone', $2::text,
        'days', (select jsonb_agg(jsonb_build_object('date', a.day, 'submissions', coalesce(d.submissions, 0),
                'firstSolves', coalesce(f.solves, 0)) order by a.day)
            from activity a left join daily d using(day) left join first_solves f using(day))),
    'streaks', (select jsonb_build_object('currentDays', coalesce(max(length) filter (where last_day >= $5::date - 1), 0),
        'longestDays', coalesce(max(length), 0)) from streak_lengths),
    'byDifficulty', coalesce((select items from breakdown_json where dimension = 'difficulty'), '[]'::jsonb),
    'byCategory', coalesce((select items from breakdown_json where dimension = 'category'), '[]'::jsonb),
    'byTag', coalesce((select items from breakdown_json where dimension = 'tag'), '[]'::jsonb),
    'byLanguage', coalesce((select jsonb_agg(jsonb_build_object('language', language, 'submissions', submissions,
        'acceptedSubmissions', accepted, 'attemptedTasks', attempted, 'solvedTasks', solved) order by language)
        from language_counts), '[]'::jsonb)
)`
