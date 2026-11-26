alter table judge_jobs
    add column result_payload json,
    add column callback_next_attempt_at timestamptz,
    add column callback_attempts bigint not null default 0;

create index judge_jobs_callback_due_idx on judge_jobs (callback_next_attempt_at, id)
    where callback_next_attempt_at is not null;
