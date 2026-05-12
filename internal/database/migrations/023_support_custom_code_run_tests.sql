begin;

alter table code_run_test_results
    alter column test_case_id drop not null,
    add column if not exists test_name text,
    add column if not exists is_custom boolean not null default false;

commit;
