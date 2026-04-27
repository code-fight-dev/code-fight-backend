begin;

alter table submissions
    add column if not exists judge_submission_id text,
    add column if not exists judge_status text,
    add column if not exists error_message text,
    add column if not exists passed_tests integer not null default 0,
    add column if not exists total_tests integer not null default 0,
    add column if not exists compile_time_ms integer,
    add column if not exists run_time_ms integer,
    add column if not exists peak_memory_kb integer,
    add column if not exists score integer not null default 0,
    add column if not exists exit_code integer,
    add column if not exists compile_log text,
    add column if not exists stdout_truncated text,
    add column if not exists stderr_truncated text,
    add column if not exists started_at timestamptz,
    add column if not exists finished_at timestamptz;

do $$
begin
    if exists (
        select 1
        from pg_constraint
        where conname = 'submissions_status_check'
          and conrelid = 'submissions'::regclass
    ) then
        alter table submissions drop constraint submissions_status_check;
    end if;

    alter table submissions
        add constraint submissions_status_check
        check (status in ('queued', 'sent_to_judge', 'running', 'finished', 'failed'));
end $$;

do $$
begin
    if exists (
        select 1
        from pg_constraint
        where conname = 'submissions_verdict_check'
          and conrelid = 'submissions'::regclass
    ) then
        alter table submissions drop constraint submissions_verdict_check;
    end if;

    alter table submissions
        add constraint submissions_verdict_check
        check (
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
        );
end $$;

do $$
begin
    if exists (
        select 1
        from pg_constraint
        where conname = 'submissions_judge_status_check'
          and conrelid = 'submissions'::regclass
    ) then
        alter table submissions drop constraint submissions_judge_status_check;
    end if;

    alter table submissions
        add constraint submissions_judge_status_check
        check (
            judge_status is null or judge_status in (
                'pending',
                'queued',
                'running',
                'finished',
                'failed'
            )
        );
end $$;

do $$
begin
    if exists (
        select 1
        from pg_constraint
        where conname = 'submissions_passed_tests_check'
          and conrelid = 'submissions'::regclass
    ) then
        alter table submissions drop constraint submissions_passed_tests_check;
    end if;

    alter table submissions
        add constraint submissions_passed_tests_check
        check (passed_tests >= 0);
end $$;

do $$
begin
    if exists (
        select 1
        from pg_constraint
        where conname = 'submissions_total_tests_check'
          and conrelid = 'submissions'::regclass
    ) then
        alter table submissions drop constraint submissions_total_tests_check;
    end if;

    alter table submissions
        add constraint submissions_total_tests_check
        check (total_tests >= 0);
end $$;

do $$
begin
    if exists (
        select 1
        from pg_constraint
        where conname = 'submissions_score_check'
          and conrelid = 'submissions'::regclass
    ) then
        alter table submissions drop constraint submissions_score_check;
    end if;

    alter table submissions
        add constraint submissions_score_check
        check (score >= 0);
end $$;

create index if not exists submissions_verdict_idx on submissions (verdict);
create index if not exists submissions_judge_status_idx on submissions (judge_status);
create unique index if not exists submissions_judge_submission_id_uidx
    on submissions (judge_submission_id)
    where judge_submission_id is not null;

commit;
