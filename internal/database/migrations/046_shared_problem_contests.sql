alter table tournaments drop constraint tournaments_status_check;
alter table tournaments
    add column format text not null default 'round_robin',
    add constraint tournaments_format_check check (format in ('round_robin', 'shared_problem_set')),
    add constraint tournaments_status_check check (
        (status = 'draft' and task_id is null and format = 'round_robin') or
        (status in ('published', 'running', 'completed') and (
            (format = 'round_robin' and task_id is not null) or
            (format = 'shared_problem_set' and task_id is null and starts_at is not null)
        ))
    );

create table tournament_problems (
    tournament_id uuid not null references tournaments(id) on delete cascade,
    task_id uuid not null references tasks(id) on delete restrict,
    position integer not null check (position > 0),
    primary key (tournament_id, task_id),
    unique (tournament_id, position)
);

alter table submissions
    add column tournament_id uuid,
    add constraint submissions_scope_check check (match_id is null or tournament_id is null),
    add foreign key (tournament_id, user_id) references tournament_participants(tournament_id, user_id),
    add foreign key (tournament_id, task_id) references tournament_problems(tournament_id, task_id);

create index submissions_contest_standings_idx on submissions (tournament_id, user_id, task_id, created_at, id)
    where tournament_id is not null;
create index submissions_contest_pending_idx on submissions (tournament_id)
    where tournament_id is not null and status in ('queued', 'sent_to_judge', 'running');

create table tournament_problem_results (
    tournament_id uuid not null,
    user_id uuid not null,
    rank integer not null check (rank > 0),
    solved integer not null check (solved >= 0),
    penalty_seconds bigint not null check (penalty_seconds >= 0),
    judged_submissions integer not null check (judged_submissions >= 0),
    primary key (tournament_id, user_id),
    foreign key (tournament_id, user_id) references tournament_participants(tournament_id, user_id) on delete cascade
);

drop trigger notifications_tournament_status on tournaments;
create trigger notifications_tournament_status
    after update of status on tournaments for each row
    when (old.status is distinct from new.status and new.format = 'round_robin')
    execute function notify_tournament_status();

create function notify_shared_contest_status() returns trigger language plpgsql as $$
begin
    if new.status = 'running' then
        insert into notifications (user_id, kind, title, message, href, event_key)
        select user_id, 'tournament.started', 'Tournament started',
               new.title || ' · The problems are ready.',
               '/tournaments/' || new.id, new.id::text
        from tournament_participants where tournament_id = new.id;
    elsif new.status = 'completed' then
        insert into notifications (user_id, kind, title, message, href, event_key)
        select user_id, 'tournament.completed', 'Tournament finished',
               new.title || ' · Place ' || rank || ' · Solved ' || solved,
               '/tournaments/' || new.id, new.id::text
        from tournament_problem_results where tournament_id = new.id;
    end if;
    return null;
end;
$$;

create trigger notifications_shared_contest_status
    after update of status on tournaments for each row
    when (old.status is distinct from new.status and new.format = 'shared_problem_set')
    execute function notify_shared_contest_status();
