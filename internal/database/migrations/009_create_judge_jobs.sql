CREATE TABLE IF NOT EXISTS judge_jobs (
    id UUID PRIMARY KEY,
    submission_id UUID NOT NULL,
    task_id UUID NOT NULL,
    language TEXT NOT NULL,
    language_version TEXT NOT NULL,
    source_code TEXT NOT NULL,
    callback_url TEXT NOT NULL,
    callback_token TEXT NOT NULL,
    time_limit_ms INTEGER NOT NULL CHECK (time_limit_ms > 0),
    memory_limit_mb INTEGER NOT NULL CHECK (memory_limit_mb > 0),
    output_limit_kb INTEGER NOT NULL CHECK (output_limit_kb > 0),
    tests_json JSONB NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_judge_jobs_status_created_at
    ON judge_jobs (status, created_at);