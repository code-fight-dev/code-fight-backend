begin;

do $$
begin
    if exists (
        select 1
        from information_schema.columns
        where table_name = 'submissions'
          and column_name = 'match_id'
          and is_nullable = 'NO'
    ) then
        alter table submissions
            alter column match_id drop not null;
    end if;
end $$;

commit;
