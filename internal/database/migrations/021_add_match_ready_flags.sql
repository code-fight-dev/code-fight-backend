alter table matches
    add column if not exists player1_ready boolean not null default false,
    add column if not exists player2_ready boolean not null default false;
