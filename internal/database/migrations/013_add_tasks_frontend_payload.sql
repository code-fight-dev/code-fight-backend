begin;

alter table tasks
    add column if not exists frontend_payload jsonb not null default '{}'::jsonb;

commit;
