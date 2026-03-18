alter table users
    add column if not exists city text not null default '';

alter table users
    add column if not exists avatar_source text not null default 'none';

alter table users
    add column if not exists provider_avatar_url text not null default '';

do $$
begin
    if not exists (
        select 1
        from pg_constraint
        where conname = 'users_avatar_source_check'
    ) then
        alter table users
            add constraint users_avatar_source_check
            check (avatar_source in ('none', 'provider', 'custom'));
    end if;
end $$;
