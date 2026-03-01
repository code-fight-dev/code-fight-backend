create table if not exists users (
    id uuid primary key,
    email text not null,
    username text not null,
    password_hash text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create unique index if not exists users_email_lower_key on users (lower(email));
create unique index if not exists users_username_lower_key on users (lower(username));

create table if not exists sessions (
    id uuid primary key,
    user_id uuid not null references users(id) on delete cascade,
    token_hash text not null unique,
    user_agent text not null default '',
    ip_address text not null default '',
    expires_at timestamptz not null,
    created_at timestamptz not null default now()
);

create index if not exists sessions_user_id_idx on sessions (user_id);
create index if not exists sessions_expires_at_idx on sessions (expires_at);
