alter table notifications
    add column event_key text,
    add column deliver_at timestamptz;

create unique index notifications_event_once_idx on notifications (user_id, kind, event_key);
create index notifications_due_idx on notifications (deliver_at) where deliver_at is not null;

create function publish_notification_change() returns trigger language plpgsql as $$
begin
    perform pg_notify('codefight_notifications', new.user_id::text);
    return null;
end;
$$;

create trigger notifications_publish_insert
    after insert on notifications for each row
    when (new.deliver_at is null) execute function publish_notification_change();
create trigger notifications_publish_delivery
    after update of deliver_at on notifications for each row
    when (old.deliver_at is not null and new.deliver_at is null) execute function publish_notification_change();

create function notify_match_cancelled() returns trigger language plpgsql as $$
begin
    if new.player1_ready <> new.player2_ready then
        insert into notifications (user_id, kind, title, message, href, event_key)
        values (
            case when new.player1_ready then new.player1_id else new.player2_id end,
            'match.cancelled', 'Match cancelled',
            'Your opponent did not confirm readiness before the match was cancelled.',
            '/arena', new.id::text
        );
    end if;

    insert into notifications (user_id, kind, title, message, href, event_key)
    select t.owner_id, 'tournament.match_cancelled', 'Tournament match cancelled',
           t.title || ' · You can schedule this pairing again.', '/tournaments/' || t.id, new.id::text
    from tournament_fixtures f join tournaments t on t.id=f.tournament_id
    where f.match_id=new.id;
    return null;
end;
$$;

create trigger notifications_match_cancelled
    after update of status on matches for each row
    when (old.status='pending' and new.status='cancelled') execute function notify_match_cancelled();

create function notify_match_finished() returns trigger language plpgsql as $$
begin
    insert into notifications (user_id, kind, title, message, href, event_key)
    select p.user_id, 'match.finished',
           case when new.winner_id is null then 'Match drawn'
                when new.winner_id=p.user_id then 'You won the match'
                else 'You lost the match' end,
           'Opponent ' || opponent.username ||
               case when r.delta is null then '' else
                    ' · Rating ' || case when r.delta >= 0 then '+' else '' end || r.delta end,
           '/arena/match/' || new.id, new.id::text
    from (values (new.player1_id,new.player2_id),(new.player2_id,new.player1_id)) p(user_id,opponent_id)
    join users opponent on opponent.id=p.opponent_id
    left join rating_history r on r.match_id=new.id and r.user_id=p.user_id and r.mode=new.rating_mode;
    return null;
end;
$$;

create trigger notifications_match_finished
    after update of rating_applied on matches for each row
    when (not old.rating_applied and new.rating_applied and new.status='finished')
    execute function notify_match_finished();

create function notify_tournament_status() returns trigger language plpgsql as $$
begin
    if new.status='running' then
        insert into notifications (user_id, kind, title, message, href, event_key)
        select user_id, 'tournament.started', 'Tournament started',
               new.title || ' · Registration has closed and pairings are ready.',
               '/tournaments/' || new.id, new.id::text
        from tournament_participants where tournament_id=new.id;
    elsif new.status='completed' then
        with standings as (
            select p.user_id, u.username,
                   count(*) filter (where m.winner_id=p.user_id) as wins,
                   count(*) filter (where m.result_type='draw') as draws
            from tournament_participants p join users u on u.id=p.user_id
            left join tournament_fixtures f on f.tournament_id=p.tournament_id
                and p.user_id in (f.player1_id,f.player2_id)
            left join matches m on m.id=f.match_id and m.status='finished'
            where p.tournament_id=new.id
            group by p.user_id,u.username
        ), ranked as (
            select user_id, 3*wins+draws as points,
                   row_number() over (order by 3*wins+draws desc,wins desc,username,user_id) as place
            from standings
        )
        insert into notifications (user_id, kind, title, message, href, event_key)
        select user_id, 'tournament.completed', 'Tournament finished',
               new.title || ' · Place ' || place || ' · Points ' || points,
               '/tournaments/' || new.id, new.id::text
        from ranked;
    end if;
    return null;
end;
$$;

create trigger notifications_tournament_status
    after update of status on tournaments for each row
    when (old.status is distinct from new.status) execute function notify_tournament_status();

create function notify_role_change() returns trigger language plpgsql as $$
begin
    insert into notifications (user_id, kind, title, message, href, event_key)
    values (new.id, 'account.role_changed', 'Account role changed',
            'Your account role changed from ' || old.role || ' to ' || new.role || '.',
            '/settings/security', new.role_version::text)
    on conflict (user_id, kind, event_key) do nothing;
    return null;
end;
$$;

create trigger notifications_account_role
    after update of role on users for each row
    when (old.role is distinct from new.role) execute function notify_role_change();

create function notify_submission_result() returns trigger language plpgsql as $$
begin
    insert into notifications (user_id, kind, title, message, href, event_key, deliver_at)
    select new.user_id, 'submission.reviewed', 'Submission checked',
           task.title || ' · ' || initcap(replace(coalesce(new.verdict, 'failed'), '_', ' ')),
           '/challenges/' || task.slug, new.id::text, clock_timestamp()+interval '15 seconds'
    from tasks task where task.id=new.task_id
    on conflict (user_id, kind, event_key) do nothing;
    return null;
end;
$$;

create trigger notifications_submission_result
    after update of status on submissions for each row
    when (new.match_id is null and old.status not in ('finished','failed') and new.status in ('finished','failed'))
    execute function notify_submission_result();
