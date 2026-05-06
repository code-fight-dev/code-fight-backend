create index if not exists submissions_user_task_created_at_idx
    on submissions (user_id, task_id, created_at desc);
