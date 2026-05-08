create table if not exists matchmaking_queue (
    user_id uuid primary key references users(id) on delete cascade,
    rating_mode text not null default 'global',
    task_mode text not null default 'normal',
    is_rated boolean not null default true,
    created_at timestamptz not null default now(),
    constraint matchmaking_queue_task_mode_check check (task_mode in ('normal', 'hard'))
);

create index if not exists matchmaking_queue_lookup_idx
    on matchmaking_queue (rating_mode, task_mode, is_rated, created_at);
