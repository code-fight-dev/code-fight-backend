package matches

const matchSelectColumns = `
	id,
	rating_mode,
	task_mode,
	task_id,
	task_difficulty_snapshot,
	player1_id,
	player2_id,
	player1_ready,
	player2_ready,
	winner_id,
	status,
	judge_status,
	result_type,
	winning_reason,
	is_rated,
	rating_applied,
	player1_score,
	player2_score,
	player1_attempts,
	player2_attempts,
	player1_solved,
	player2_solved,
	player1_solved_at,
	player2_solved_at,
	started_at,
	finished_at,
	expires_at,
	created_at,
	updated_at
`

const findQueuedOpponentQuery = `
	with requester as (
		select
			coalesce((
				select rating
				from user_ratings
				where user_id = $1
				  and mode = $2
			), 1000) as rating,
			coalesce((
				select created_at
				from matchmaking_queue
				where user_id = $1
				  and rating_mode = $2
				  and task_mode = $3
				  and is_rated = $4
			), now()) as queued_at
	)
	select q.user_id
	from matchmaking_queue q
	cross join requester
	left join user_ratings opponent_rating
	  on opponent_rating.user_id = q.user_id
	 and opponent_rating.mode = q.rating_mode
	where q.user_id <> $1
	  and q.rating_mode = $2
	  and q.task_mode = $3
	  and q.is_rated = $4
	  and abs(coalesce(opponent_rating.rating, 1000) - requester.rating) <= least(
		case
			when now() - q.created_at < interval '1 minute' then 60
			when now() - q.created_at < interval '3 minutes' then 120
			when now() - q.created_at < interval '5 minutes' then 160
			else 250
		end,
		case
			when now() - requester.queued_at < interval '1 minute' then 60
			when now() - requester.queued_at < interval '3 minutes' then 120
			when now() - requester.queued_at < interval '5 minutes' then 160
			else 250
		end
	  )
	order by abs(coalesce(opponent_rating.rating, 1000) - requester.rating) asc,
	         q.created_at asc
	limit 1
`

const enqueuePlayerQuery = `
	insert into matchmaking_queue (user_id, rating_mode, task_mode, is_rated, created_at)
	values ($1, $2, $3, $4, now())
	on conflict (user_id) do update set
		rating_mode = excluded.rating_mode,
		task_mode = excluded.task_mode,
		is_rated = excluded.is_rated,
		created_at = case
			when matchmaking_queue.rating_mode = excluded.rating_mode
			 and matchmaking_queue.task_mode = excluded.task_mode
			 and matchmaking_queue.is_rated = excluded.is_rated
			then matchmaking_queue.created_at
			else now()
		end
`

const getQueuedPlayerQuery = `
	select user_id, rating_mode, task_mode, is_rated, created_at
	from matchmaking_queue
	where user_id = $1
`

const removeQueuedPlayersQuery = `
	delete from matchmaking_queue
	where user_id = any($1)
`

const publicStatsQuery = `
	with active_players as (
		select player1_id as user_id
		from matches
		where status in ('pending', 'running')
		union
		select player2_id as user_id
		from matches
		where status in ('pending', 'running')
	)
	select
		(select count(*)::int from matchmaking_queue) as queued_players,
		(select count(*)::int from active_players) as active_match_players
`

const createMatchQuery = `
	insert into matches (
		id,
		rating_mode,
		task_mode,
		task_id,
		task_difficulty_snapshot,
		player1_id,
		player2_id,
		player1_ready,
		player2_ready,
		winner_id,
		status,
		judge_status,
		result_type,
		winning_reason,
		is_rated,
		rating_applied,
		player1_score,
		player2_score,
		player1_attempts,
		player2_attempts,
		player1_solved,
		player2_solved,
		player1_solved_at,
		player2_solved_at,
		started_at,
		finished_at,
		expires_at,
		created_at,
		updated_at
	)
	values (
		$1, $2, $3, $4, $5,
		$6, $7, $8, $9, $10,
		$11, $12, $13, $14, $15,
		$16, $17, $18, $19, $20,
		$21, $22, $23, $24, $25,
		$26, $27, now(), now()
	)
	returning ` + matchSelectColumns

const getMatchQuery = `
	select ` + matchSelectColumns + `
	from matches
	where id = $1
`

const acceptMatchQuery = `
	update matches
	set player1_ready = $2,
		player2_ready = $3,
		updated_at = now()
	where id = $1
	returning ` + matchSelectColumns

const startMatchQuery = `
	update matches
	set status = $2,
		judge_status = $3,
		player1_ready = true,
		player2_ready = true,
		started_at = now(),
		expires_at = now() + ($4::text)::interval,
		updated_at = now()
	where id = $1
	returning ` + matchSelectColumns

const cancelMatchQuery = `
	update matches
	set status = $2,
		result_type = $3,
		winning_reason = $4,
		judge_status = $5,
		finished_at = coalesce(finished_at, now()),
		updated_at = now()
	where id = $1
`

const listUserMatchesQuery = `
	select ` + matchSelectColumns + `
	from matches
	where player1_id = $1 or player2_id = $1
	order by created_at desc
	limit $2 offset $3
`

const getCurrentMatchQuery = `
	select ` + matchSelectColumns + `
	from matches
	where (player1_id = $1 or player2_id = $1)
	  and status in ('pending', 'running')
	order by
		case when status = 'running' then 0 else 1 end,
		created_at desc
	limit 1
`

const listMatchSubmissionsQuery = `
	select
		id,
		match_id,
		task_id,
		user_id,
		language,
		language_version,
		status,
		verdict,
		passed_tests,
		total_tests,
		score,
		created_at,
		started_at,
		finished_at,
		updated_at
	from submissions
	where match_id = $1
	order by created_at desc
	limit 100
`

const listExpiredMatchesQuery = `
	select id
	from matches
	where status = 'running'
	  and expires_at is not null
	  and expires_at <= $1
	order by expires_at asc
	limit $2
`

const cancelExpiredPendingMatchesQuery = `
	with expired as (
		select id
		from matches
		where status = 'pending'
		  and created_at <= now() - ($1::text)::interval
		order by created_at asc
		limit $2
		for update skip locked
	)
	update matches m
	set status = $3,
		result_type = $4,
		winning_reason = $5,
		judge_status = $6,
		finished_at = coalesce(m.finished_at, now()),
		updated_at = now()
	from expired
	where m.id = expired.id
`

const finishMatchQuery = `
	update matches
	set winner_id = $2,
		status = $3,
		result_type = $4,
		winning_reason = $5,
		judge_status = $6,
		finished_at = $7,
		updated_at = now()
	where id = $1
`

const markMatchRatingAppliedQuery = `
	update matches
	set rating_applied = true,
		updated_at = now()
	where id = $1
`

const getMatchForUpdateQuery = `
	select ` + matchSelectColumns + `
	from matches
	where id = $1
	for update
`

const getUserRatingForUpdateQuery = `
	select
		user_id, mode, rating, rated_games, wins, losses, draws, created_at, updated_at
	from user_ratings
	where user_id = $1 and mode = $2
	for update
`

const updateUserRatingAfterMatchQuery = `
	update user_ratings
	set rating = $3,
		rated_games = rated_games + 1,
		wins = $4,
		losses = $5,
		draws = $6,
		updated_at = now()
	where user_id = $1 and mode = $2
`

const insertRatingHistoryQuery = `
	insert into rating_history (
		id, user_id, match_id, mode, old_rating, new_rating, delta, created_at
	)
	values ($1, $2, $3, $4, $5, $6, $7, now())
`

const applyPlayer1SubmissionProgressQuery = `
	update matches
	set
		player1_attempts = player1_attempts + 1,
		player1_score = greatest(player1_score, $2),
		player1_solved = case
			when $3 then true
			else player1_solved
		end,
		player1_solved_at = case
			when $3 and player1_solved_at is null then $4
			else player1_solved_at
		end,
		judge_status = coalesce($5, judge_status)
	where id = $1
`

const applyPlayer2SubmissionProgressQuery = `
	update matches
	set
		player2_attempts = player2_attempts + 1,
		player2_score = greatest(player2_score, $2),
		player2_solved = case
			when $3 then true
			else player2_solved
		end,
		player2_solved_at = case
			when $3 and player2_solved_at is null then $4
			else player2_solved_at
		end,
		judge_status = coalesce($5, judge_status)
	where id = $1
`
