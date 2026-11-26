create table judge_dispatch (
    id uuid primary key,
    submission_id uuid unique references submissions(id) on delete cascade,
    code_run_id uuid unique references code_runs(id) on delete cascade,
    payload jsonb not null,
    attempts integer not null default 0,
    next_attempt_at timestamptz not null default now(),
    check (
        (submission_id is not null and submission_id = id and code_run_id is null)
        or (code_run_id is not null and code_run_id = id and submission_id is null)
    )
);

create index judge_dispatch_ready_idx on judge_dispatch (next_attempt_at, id);
create index judge_jobs_submission_lookup_idx on judge_jobs (submission_id, created_at, id);
