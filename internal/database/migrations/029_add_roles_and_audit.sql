alter table users
    add column role text not null default 'user'
        check (role in ('user', 'organizer', 'admin')),
    add column role_version bigint not null default 1 check (role_version > 0);

create table audit_events (
    id uuid primary key,
    actor_type text not null check (actor_type in ('user', 'system')),
    actor_id uuid,
    action text not null check (length(action) between 1 and 100),
    entity_type text not null,
    entity_id uuid not null,
    tournament_id uuid,
    request_id text not null,
    metadata jsonb not null default '{}'::jsonb check (jsonb_typeof(metadata) = 'object'),
    occurred_at timestamptz not null default clock_timestamp(),
    constraint audit_actor_check check (
        (actor_type = 'user' and actor_id is not null)
        or (actor_type = 'system' and actor_id is null)
    )
);

create index audit_events_entity_idx on audit_events (entity_type, entity_id, occurred_at desc, id);
create index audit_events_tournament_idx on audit_events (tournament_id, occurred_at desc, id)
    where tournament_id is not null;

create function reject_audit_mutation() returns trigger language plpgsql as $$
begin
    raise exception 'audit_events is append-only' using errcode = '42501';
end;
$$;

create trigger audit_events_immutable
    before update or delete on audit_events
    for each row execute function reject_audit_mutation();
create trigger audit_events_no_truncate
    before truncate on audit_events
    for each statement execute function reject_audit_mutation();
revoke update, delete, truncate on audit_events from public;
