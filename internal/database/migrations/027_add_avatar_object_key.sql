alter table users
    add column if not exists avatar_object_key text not null default '';
