create index if not exists submissions_task_accepted_idx
    on submissions (task_id)
    where status = 'finished'
      and verdict = 'accepted';
