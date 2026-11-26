create function tournament_match_waits_for_judge(match_id uuid, cutoff timestamptz)
returns boolean language sql stable as $$
    select exists (
        select 1 from submissions s
        where s.match_id = $1 and s.created_at < $2 and (
            s.status in ('queued', 'sent_to_judge', 'running')
            or (s.status = 'finished' and s.verdict = 'accepted')
            or (s.match_progress_applied_at is null and (
                s.status = 'finished' or (s.status = 'failed' and s.started_at is not null)
            ))
        )
    )
$$;
