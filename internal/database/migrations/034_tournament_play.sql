alter table tournaments drop constraint tournaments_status_check;
alter table tournaments
    add column task_id uuid references tasks(id) on delete restrict,
    add constraint tournaments_status_check check (
        (status = 'draft' and task_id is null) or
        (status in ('published', 'running', 'completed') and task_id is not null)
    );

create table tournament_participants (
    tournament_id uuid not null references tournaments(id) on delete cascade,
    user_id uuid not null references users(id) on delete restrict,
    primary key (tournament_id, user_id)
);

create table tournament_fixtures (
    id uuid primary key default gen_random_uuid(),
    tournament_id uuid not null references tournaments(id) on delete cascade,
    player1_id uuid not null,
    player2_id uuid not null,
    match_id uuid unique references matches(id) on delete restrict,
    check (player1_id < player2_id),
    unique (tournament_id, player1_id, player2_id),
    foreign key (tournament_id, player1_id) references tournament_participants(tournament_id, user_id),
    foreign key (tournament_id, player2_id) references tournament_participants(tournament_id, user_id)
);
