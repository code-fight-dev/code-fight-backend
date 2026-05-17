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
