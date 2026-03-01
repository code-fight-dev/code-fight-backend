create table if not exists oauth_accounts (
    id uuid primary key,
    user_id uuid not null references users(id) on delete cascade,
    provider text not null,
    provider_user_id text not null,
    provider_email text not null default '',
    created_at timestamptz not null default now(),
    constraint oauth_accounts_provider_external_key unique (provider, provider_user_id),
    constraint oauth_accounts_provider_user_key unique (provider, user_id)
);

create index if not exists oauth_accounts_user_id_idx on oauth_accounts (user_id);
