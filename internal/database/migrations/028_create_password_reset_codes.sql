create table if not exists password_reset_codes (
    id uuid primary key,
    user_id uuid not null references users(id) on delete cascade,
    code_hash text not null,
    expires_at timestamptz not null,
    used_at timestamptz,
    attempts int not null default 0,
    created_at timestamptz not null default now()
);

create index if not exists password_reset_codes_user_id_idx on password_reset_codes (user_id);
create index if not exists password_reset_codes_expires_at_idx on password_reset_codes (expires_at);
