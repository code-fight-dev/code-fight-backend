create table notifications (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references users(id) on delete cascade,
    kind text not null,
    title text not null,
    message text not null,
    href text not null check (left(href, 1) = '/' and left(href, 2) <> '//' and strpos(href, chr(92)) = 0),
    deadline_at timestamptz,
    created_at timestamptz not null default clock_timestamp(),
    read_at timestamptz
);

create index notifications_user_history_idx on notifications (user_id, created_at desc, id desc);
create index notifications_user_unread_idx on notifications (user_id) where read_at is null;
