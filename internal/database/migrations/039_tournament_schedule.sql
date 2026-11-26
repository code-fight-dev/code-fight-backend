alter table tournaments
    add column registration_closes_at timestamptz,
    add column starts_at timestamptz,
    add column ends_at timestamptz,
    add column completed_at timestamptz,
    add constraint tournament_schedule_order_check check (
        (registration_closes_at is null and starts_at is null and ends_at is null)
        or (registration_closes_at is not null and starts_at is not null and ends_at is not null
            and registration_closes_at <= starts_at and starts_at < ends_at)
    );

alter table tournament_participants add column joined_at timestamptz;
alter table tournament_participants alter column joined_at set default clock_timestamp();

create index tournament_scheduled_open_idx on tournaments (starts_at, id)
    where starts_at is not null and status in ('published', 'running');
create index tournament_participation_by_user_idx on tournament_participants (user_id, tournament_id);
