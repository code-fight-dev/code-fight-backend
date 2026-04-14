begin;

do $$
begin
    if exists (
        select 1
        from information_schema.columns
        where table_name = 'matches'
          and column_name = 'mode'
    ) and not exists (
        select 1
        from information_schema.columns
        where table_name = 'matches'
          and column_name = 'rating_mode'
    ) then
        alter table matches rename column mode to rating_mode;
    end if;
end $$;

drop index if exists matches_mode_idx;
create index if not exists matches_rating_mode_idx on matches (rating_mode);

create table if not exists tasks (
    id uuid primary key,
    slug text not null unique,
    title text not null,
    statement_md text not null,
    difficulty text not null,
    time_limit_ms integer not null,
    memory_limit_kb integer not null,
    is_active boolean not null default true,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint tasks_difficulty_check check (difficulty in ('easy', 'medium', 'hard')),
    constraint tasks_time_limit_check check (time_limit_ms > 0),
    constraint tasks_memory_limit_check check (memory_limit_kb > 0)
);

create index if not exists tasks_difficulty_idx on tasks (difficulty);
create index if not exists tasks_is_active_idx on tasks (is_active);
create index if not exists tasks_difficulty_is_active_idx on tasks (difficulty, is_active);

alter table matches
    add column if not exists task_mode text not null default 'normal',
    add column if not exists task_id uuid,
    add column if not exists task_difficulty_snapshot text,
    add column if not exists judge_status text not null default 'not_started',
    add column if not exists player1_score integer not null default 0,
    add column if not exists player2_score integer not null default 0,
    add column if not exists player1_attempts integer not null default 0,
    add column if not exists player2_attempts integer not null default 0,
    add column if not exists player1_solved boolean not null default false,
    add column if not exists player2_solved boolean not null default false,
    add column if not exists player1_solved_at timestamptz,
    add column if not exists player2_solved_at timestamptz,
    add column if not exists result_type text,
    add column if not exists winning_reason text;

do $$
begin
    if not exists (
        select 1 from pg_constraint
        where conname = 'matches_task_mode_check'
    ) then
        alter table matches
            add constraint matches_task_mode_check
            check (task_mode in ('normal', 'hard'));
    end if;
end $$;

do $$
begin
    if not exists (
        select 1 from pg_constraint
        where conname = 'matches_judge_status_check'
    ) then
        alter table matches
            add constraint matches_judge_status_check
            check (judge_status in ('not_started', 'ready', 'running', 'finished', 'failed'));
    end if;
end $$;

do $$
begin
    if not exists (
        select 1 from pg_constraint
        where conname = 'matches_task_difficulty_snapshot_check'
    ) then
        alter table matches
            add constraint matches_task_difficulty_snapshot_check
            check (
                task_difficulty_snapshot is null
                or task_difficulty_snapshot in ('easy', 'medium', 'hard')
            );
    end if;
end $$;

do $$
begin
    if not exists (
        select 1 from pg_constraint
        where conname = 'matches_result_type_check'
    ) then
        alter table matches
            add constraint matches_result_type_check
            check (
                result_type is null
                or result_type in ('player1_win', 'player2_win', 'draw', 'cancelled')
            );
    end if;
end $$;

do $$
begin
    if not exists (
        select 1 from pg_constraint
        where conname = 'matches_winning_reason_check'
    ) then
        alter table matches
            add constraint matches_winning_reason_check
            check (
                winning_reason is null
                or winning_reason in (
                    'accepted_faster',
                    'accepted_more_tests',
                    'opponent_failed',
                    'draw',
                    'cancelled'
                )
            );
    end if;
end $$;

do $$
begin
    if not exists (
        select 1 from pg_constraint
        where conname = 'matches_task_id_fkey'
    ) then
        alter table matches
            add constraint matches_task_id_fkey
            foreign key (task_id) references tasks(id) on delete set null;
    end if;
end $$;

create index if not exists matches_task_mode_idx on matches (task_mode);
create index if not exists matches_task_id_idx on matches (task_id);
create index if not exists matches_judge_status_idx on matches (judge_status);

create table if not exists task_languages (
    id uuid primary key,
    task_id uuid not null references tasks(id) on delete cascade,
    language text not null,
    language_version text not null,
    is_enabled boolean not null default true,
    created_at timestamptz not null default now(),
    unique (task_id, language, language_version)
);

create index if not exists task_languages_task_id_idx on task_languages (task_id);
create index if not exists task_languages_language_idx on task_languages (language);

create table if not exists task_test_cases (
    id uuid primary key,
    task_id uuid not null references tasks(id) on delete cascade,
    kind text not null default 'hidden',
    order_no integer not null,
    input_blob_ref text not null,
    expected_blob_ref text not null,
    weight integer not null default 1,
    is_active boolean not null default true,
    created_at timestamptz not null default now(),
    constraint task_test_cases_kind_check check (kind in ('sample', 'hidden')),
    constraint task_test_cases_order_no_check check (order_no >= 0),
    constraint task_test_cases_weight_check check (weight > 0),
    unique (task_id, order_no)
);

create index if not exists task_test_cases_task_id_idx on task_test_cases (task_id);
create index if not exists task_test_cases_task_id_kind_idx on task_test_cases (task_id, kind);

create table if not exists submissions (
    id uuid primary key,
    match_id uuid references matches(id),
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
    constraint submissions_status_check check (
        status in ('queued', 'sent_to_judge', 'running', 'finished', 'failed')
    ),
    constraint submissions_verdict_check check (
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
    constraint submissions_judge_status_check check (
        judge_status is null or judge_status in (
            'pending',
            'queued',
            'running',
            'finished',
            'failed'
        )
    ),
    constraint submissions_passed_tests_check check (passed_tests >= 0),
    constraint submissions_total_tests_check check (total_tests >= 0),
    constraint submissions_score_check check (score >= 0)
);

create index if not exists submissions_match_id_idx on submissions (match_id);
create index if not exists submissions_task_id_idx on submissions (task_id);
create index if not exists submissions_user_id_idx on submissions (user_id);
create index if not exists submissions_status_idx on submissions (status);
create index if not exists submissions_verdict_idx on submissions (verdict);
create index if not exists submissions_judge_status_idx on submissions (judge_status);
create unique index if not exists submissions_judge_submission_id_uidx
    on submissions (judge_submission_id)
    where judge_submission_id is not null;
create index if not exists submissions_match_user_created_at_idx
    on submissions (match_id, user_id, created_at desc);

create table if not exists submission_test_results (
    id uuid primary key,
    submission_id uuid not null references submissions(id) on delete cascade,
    test_case_id uuid not null references task_test_cases(id),
    status text not null,
    time_ms integer,
    memory_kb integer,
    exit_code integer,
    checker_message text,
    created_at timestamptz not null default now(),
    constraint submission_test_results_status_check check (
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
    constraint submission_test_results_submission_test_unique
        unique (submission_id, test_case_id)
);

create index if not exists submission_test_results_submission_id_idx
    on submission_test_results (submission_id);

create index if not exists submission_test_results_test_case_id_idx
    on submission_test_results (test_case_id);

commit;