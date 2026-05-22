begin;

create table if not exists match_replay_sessions (
    match_id uuid not null references matches(id) on delete cascade,
    user_id uuid not null references users(id) on delete cascade,
    version integer not null default 1,
    last_seq bigint not null default 0,
    last_t_ms bigint not null default 0,
    event_count bigint not null default 0,
    snapshot_count bigint not null default 0,
    duration_ms bigint not null default 0,
    started_at timestamptz not null default now(),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint match_replay_sessions_pk primary key (match_id, user_id),
    constraint match_replay_sessions_non_negative check (
        last_seq >= 0
        and last_t_ms >= 0
        and event_count >= 0
        and snapshot_count >= 0
        and duration_ms >= 0
    )
);

create index if not exists match_replay_sessions_user_id_idx
    on match_replay_sessions (user_id);

create table if not exists match_replay_events (
    id bigserial primary key,
    match_id uuid not null references matches(id) on delete cascade,
    user_id uuid not null references users(id) on delete cascade,
    seq bigint not null,
    t_ms bigint not null,
    event_type text not null,
    payload jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now(),
    constraint match_replay_events_seq_positive check (seq > 0),
    constraint match_replay_events_t_ms_non_negative check (t_ms >= 0),
    constraint match_replay_events_type_non_empty check (length(trim(event_type)) > 0),
    constraint match_replay_events_match_user_seq_unique unique (match_id, user_id, seq)
);

create index if not exists match_replay_events_match_seq_idx
    on match_replay_events (match_id, seq);

create index if not exists match_replay_events_match_user_seq_idx
    on match_replay_events (match_id, user_id, seq);

create index if not exists match_replay_events_match_timeline_idx
    on match_replay_events (match_id, t_ms, seq, id);

create table if not exists match_replay_snapshots (
    id bigserial primary key,
    match_id uuid not null references matches(id) on delete cascade,
    user_id uuid not null references users(id) on delete cascade,
    seq bigint not null,
    t_ms bigint not null,
    language text not null,
    source_code text not null,
    created_at timestamptz not null default now(),
    constraint match_replay_snapshots_seq_non_negative check (seq >= 0),
    constraint match_replay_snapshots_t_ms_non_negative check (t_ms >= 0),
    constraint match_replay_snapshots_language_non_empty check (length(trim(language)) > 0),
    constraint match_replay_snapshots_match_user_seq_unique unique (match_id, user_id, seq)
);

create index if not exists match_replay_snapshots_match_seq_idx
    on match_replay_snapshots (match_id, seq);

create index if not exists match_replay_snapshots_match_user_seq_idx
    on match_replay_snapshots (match_id, user_id, seq);

create index if not exists match_replay_snapshots_match_timeline_idx
    on match_replay_snapshots (match_id, t_ms, seq, id);

commit;
