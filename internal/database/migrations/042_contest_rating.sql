create table contest_ratings (
    user_id uuid primary key references users(id) on delete cascade,
    rating integer not null
);

create table contest_rating_events (
    tournament_id uuid primary key references tournaments(id) on delete cascade,
    applied_at timestamptz not null default clock_timestamp()
);

create table contest_rating_changes (
    tournament_id uuid not null references contest_rating_events(tournament_id) on delete cascade,
    user_id uuid not null references users(id) on delete cascade,
    rating_before integer not null,
    rating_after integer not null,
    primary key (tournament_id, user_id)
);
