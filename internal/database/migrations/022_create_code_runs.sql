begin;

create table if not exists code_runs (
    id uuid primary key,
    task_id uuid not null references tasks(id),
    user_id uuid not null references users(id),
    language text not null,
    language_version text not null,
    source_code text not null,
    status text not null default 'queued',
    verdict text,
    judge_submission_id text,
    judge_status text,
    error_message text,
    passed_tests integer not null default 0,
    total_tests integer not null default 0,
    compile_time_ms integer,
    run_time_ms integer,
    peak_memory_kb integer,
    score integer not null default 0,
    exit_code integer,
    compile_log text,
    stdout_truncated text,
    stderr_truncated text,
    created_at timestamptz not null default now(),
    started_at timestamptz,
    finished_at timestamptz,
    updated_at timestamptz not null default now(),
    constraint code_runs_status_check check (
        status in ('queued', 'sent_to_judge', 'running', 'finished', 'failed')
    ),
    constraint code_runs_verdict_check check (
        verdict is null or verdict in (
            'accepted',
            'wrong_answer',
            'time_limit_exceeded',
            'memory_limit_exceeded',
            'runtime_error',
            'compile_error',
            'presentation_error',
            'system_error'
        )
    ),
    constraint code_runs_judge_status_check check (
        judge_status is null or judge_status in (
            'pending',
            'queued',
            'running',
            'finished',
            'failed'
        )
    ),
    constraint code_runs_passed_tests_check check (passed_tests >= 0),
    constraint code_runs_total_tests_check check (total_tests >= 0),
    constraint code_runs_score_check check (score >= 0)
);

create index if not exists code_runs_task_id_idx on code_runs (task_id);
create index if not exists code_runs_user_id_idx on code_runs (user_id);
create index if not exists code_runs_status_idx on code_runs (status);
create index if not exists code_runs_user_task_created_at_idx
    on code_runs (user_id, task_id, created_at desc);
create unique index if not exists code_runs_judge_submission_id_uidx
    on code_runs (judge_submission_id)
    where judge_submission_id is not null;

create table if not exists code_run_test_results (
    id uuid primary key,
    code_run_id uuid not null references code_runs(id) on delete cascade,
    test_case_id uuid not null references task_test_cases(id),
    status text not null,
    time_ms integer,
    memory_kb integer,
    exit_code integer,
    checker_message text,
    created_at timestamptz not null default now(),
    constraint code_run_test_results_status_check check (
        status in (
            'accepted',
            'wrong_answer',
            'time_limit_exceeded',
            'memory_limit_exceeded',
            'runtime_error',
            'skipped',
            'system_error'
        )
    ),
    constraint code_run_test_results_run_test_unique
        unique (code_run_id, test_case_id)
);

create index if not exists code_run_test_results_code_run_id_idx
    on code_run_test_results (code_run_id);

create index if not exists code_run_test_results_test_case_id_idx
    on code_run_test_results (test_case_id);

commit;
