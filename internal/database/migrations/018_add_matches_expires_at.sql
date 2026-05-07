alter table matches
    add column if not exists expires_at timestamptz;

create index if not exists matches_status_expires_at_idx
    on matches (status, expires_at)
    where expires_at is not null;
