do $$
begin
    if exists (
        select 1 from pg_constraint
        where conname = 'matches_winning_reason_check'
    ) then
        alter table matches drop constraint matches_winning_reason_check;
    end if;

    alter table matches
        add constraint matches_winning_reason_check
        check (
            winning_reason is null
            or winning_reason in (
                'accepted_faster',
                'accepted_more_tests',
                'opponent_failed',
                'surrender',
                'draw',
                'cancelled'
            )
        );
end $$;
