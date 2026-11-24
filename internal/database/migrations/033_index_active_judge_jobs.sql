CREATE INDEX IF NOT EXISTS idx_submissions_active_judge_user
    ON submissions (user_id, updated_at)
    WHERE status IN ('queued', 'sent_to_judge', 'running');

CREATE INDEX IF NOT EXISTS idx_code_runs_active_judge_user
    ON code_runs (user_id, updated_at)
    WHERE status IN ('queued', 'sent_to_judge', 'running');
