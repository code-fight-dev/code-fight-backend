create index if not exists matches_rating_mode_created_at_idx
    on matches (rating_mode, created_at);

create index if not exists matches_status_created_at_idx
    on matches (status, created_at);
