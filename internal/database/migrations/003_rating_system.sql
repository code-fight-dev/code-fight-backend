create table if not exists user_ratings (
    user_id uuid not null references users(id) on delete cascade,
    mode text not null default 'global',
    rating integer not null default 1000,
    rated_games integer not null default 0,
    wins integer not null default 0,
    losses integer not null default 0,
    draws integer not null default 0,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    primary key (user_id, mode)
);

create index if not exists user_ratings_mode_rating_idx
    on user_ratings (mode, rating desc);

create table if not exists matches (
    id uuid primary key,
    mode text not null default 'global',
    player1_id uuid not null references users(id) on delete cascade,
    player2_id uuid not null references users(id) on delete cascade,
    winner_id uuid references users(id) on delete set null,
    status text not null default 'pending',
    is_rated boolean not null default true,
    rating_applied boolean not null default false,
    started_at timestamptz,
    finished_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint matches_players_not_same check (player1_id <> player2_id),
    constraint matches_status_check check (status in ('pending', 'running', 'finished', 'cancelled'))
);

create index if not exists matches_player1_id_idx on matches (player1_id);
create index if not exists matches_player2_id_idx on matches (player2_id);
create index if not exists matches_winner_id_idx on matches (winner_id);
create index if not exists matches_status_idx on matches (status);
create index if not exists matches_mode_idx on matches (mode);
create index if not exists matches_finished_at_idx on matches (finished_at);

create table if not exists rating_history (
    id uuid primary key,
    user_id uuid not null references users(id) on delete cascade,
    match_id uuid not null references matches(id) on delete cascade,
    mode text not null default 'global',
    old_rating integer not null,
    new_rating integer not null,
    delta integer not null,
    created_at timestamptz not null default now()
);

create index if not exists rating_history_user_id_idx on rating_history (user_id);
create index if not exists rating_history_match_id_idx on rating_history (match_id);
create index if not exists rating_history_mode_idx on rating_history (mode);
create index if not exists rating_history_created_at_idx on rating_history (created_at);