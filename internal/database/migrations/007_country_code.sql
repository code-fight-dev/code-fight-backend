alter table users
    add column if not exists country_code text not null default '';

create index if not exists users_country_code_lower_idx on users (lower(country_code));
