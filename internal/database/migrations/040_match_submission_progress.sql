alter table submissions add column match_progress_applied_at timestamptz;

-- Preserve the aggregate counters on matches that predate per-submission
-- tracking. A retry can still finish a match from its saved terminal verdict.
update submissions
set match_progress_applied_at = coalesce(finished_at, updated_at)
where match_id is not null and status in ('finished', 'failed');
