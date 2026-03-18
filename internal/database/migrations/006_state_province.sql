alter table users
    add column if not exists state_province text not null default '';
