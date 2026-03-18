alter table users
    add column if not exists display_name text not null default '';

update users
set display_name = username
where display_name = '';
