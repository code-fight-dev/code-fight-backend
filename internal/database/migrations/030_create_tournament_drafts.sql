create table tournaments (
    id uuid primary key,
    slug text not null unique check (
        length(slug) between 3 and 100 and slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'
    ),
    title text not null check (length(btrim(title)) between 1 and 200),
    description text not null default '' check (length(description) <= 20000),
    owner_id uuid not null references users(id) on delete restrict,
    created_by uuid not null references users(id) on delete restrict,
    status text not null default 'draft',
    config_version bigint not null default 1 check (config_version > 0),
    created_at timestamptz not null default clock_timestamp(),
    updated_at timestamptz not null default clock_timestamp(),
    constraint tournaments_status_check check (status = 'draft')
);
create index tournaments_owner_created_idx on tournaments (owner_id, created_at desc, id);

create function protect_tournament_identity() returns trigger language plpgsql as $$
begin
    if new.id is distinct from old.id
       or new.owner_id is distinct from old.owner_id
       or new.created_by is distinct from old.created_by
       or new.created_at is distinct from old.created_at then
        raise exception 'tournament identity and ownership are immutable' using errcode = '23514';
    end if;
    return new;
end;
$$;
create trigger tournaments_immutable_identity
    before update on tournaments
    for each row execute function protect_tournament_identity();
