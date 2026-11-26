create index submissions_practice_user_history_idx
    on submissions (user_id, created_at desc, id desc)
    where match_id is null;
