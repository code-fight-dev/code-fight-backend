alter table tournaments drop constraint tournaments_format_check, drop constraint tournaments_status_check;
alter table tournaments
    add constraint tournaments_format_check check (format in ('round_robin', 'shared_problem_set', 'single_elimination')),
    add constraint tournaments_status_check check (
        (status = 'draft' and task_id is null and format = 'round_robin') or
        (status in ('published', 'running', 'completed') and (
            (format = 'round_robin' and task_id is not null) or
            (format = 'shared_problem_set' and task_id is null and starts_at is not null) or
            (format = 'single_elimination' and task_id is not null and starts_at is not null)
        ))
    );

alter table tournament_fixtures add unique (tournament_id, id);
alter table tournament_participants add column bracket_rank integer check (bracket_rank > 0);

create table tournament_bracket_slots (
    id uuid primary key default gen_random_uuid(),
    tournament_id uuid not null references tournaments(id) on delete cascade,
    round_number integer not null check (round_number >= 1),
    position integer not null check (position >= 1),
    player1_id uuid,
    player2_id uuid,
    winner_id uuid,
    fixture_id uuid unique,
    unique (tournament_id, round_number, position),
    check (player1_id is null or player2_id is null or player1_id <> player2_id),
    check (winner_id is null or winner_id is not distinct from player1_id or winner_id is not distinct from player2_id),
    foreign key (tournament_id, player1_id) references tournament_participants(tournament_id, user_id),
    foreign key (tournament_id, player2_id) references tournament_participants(tournament_id, user_id),
    foreign key (tournament_id, fixture_id) references tournament_fixtures(tournament_id, id)
);

create function notify_single_elimination_status() returns trigger language plpgsql as $$
begin
    if new.status = 'running' then
        insert into notifications (user_id, kind, title, message, href, event_key)
        select user_id, 'tournament.started', 'Tournament started',
               new.title || ' · The bracket is ready.',
               '/tournaments/' || new.id, new.id::text
        from tournament_participants where tournament_id = new.id;
    elsif new.status = 'completed' then
        insert into notifications (user_id, kind, title, message, href, event_key)
        select user_id, 'tournament.completed', 'Tournament finished',
               new.title || ' · See the final bracket.',
               '/tournaments/' || new.id, new.id::text
        from tournament_participants where tournament_id = new.id;
    end if;
    return null;
end;
$$;

create trigger notifications_single_elimination_status
    after update of status on tournaments for each row
    when (old.status is distinct from new.status and new.format = 'single_elimination')
    execute function notify_single_elimination_status();
