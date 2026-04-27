begin;

insert into tasks (
    id,
    slug,
    title,
    statement_md,
    difficulty,
    time_limit_ms,
    memory_limit_kb,
    is_active,
    created_at,
    updated_at
) values (
    '2db3f0fb-9c15-4f58-9f59-84f74ef5f31f',
    'signal-stabilizer',
    'Signal Stabilizer',
    $$A monitoring node receives a stream of signed sensor adjustments. Starting from zero, apply each adjustment in order and track the running total.

Return the zero-based index of the first adjustment after which the running total is inside the inclusive safe band.
If the stream never stabilizes, return -1.$$,
    'easy',
    2000,
    262144,
    true,
    now(),
    now()
)
on conflict (slug) do update
set
    title = excluded.title,
    statement_md = excluded.statement_md,
    difficulty = excluded.difficulty,
    time_limit_ms = excluded.time_limit_ms,
    memory_limit_kb = excluded.memory_limit_kb,
    is_active = true,
    updated_at = now();

with selected_task as (
    select id
    from tasks
    where slug = 'signal-stabilizer'
)
insert into task_languages (
    id,
    task_id,
    language,
    language_version,
    is_enabled,
    created_at
)
select
    '6f0f6edf-3f44-42ec-98d5-ff95c574fb76',
    selected_task.id,
    'go',
    '1.24',
    true,
    now()
from selected_task
on conflict (task_id, language, language_version) do update
set
    is_enabled = true;

with selected_task as (
    select id
    from tasks
    where slug = 'signal-stabilizer'
)
insert into task_test_cases (
    id,
    task_id,
    kind,
    order_no,
    input_blob_ref,
    expected_blob_ref,
    weight,
    is_active,
    created_at
)
select
    '2eaf0d52-8af1-4b9b-bd26-a1f5cd9a1826',
    selected_task.id,
    'sample',
    1,
    '[-2, 3]
[5, -7, 1, 2]',
    '2',
    1,
    true,
    now()
from selected_task
on conflict (task_id, order_no) do update
set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;

with selected_task as (
    select id
    from tasks
    where slug = 'signal-stabilizer'
)
insert into task_test_cases (
    id,
    task_id,
    kind,
    order_no,
    input_blob_ref,
    expected_blob_ref,
    weight,
    is_active,
    created_at
)
select
    '8d04bcb5-57fc-4f98-bf9b-a4b2da0bd16e',
    selected_task.id,
    'sample',
    2,
    '[4, 6]
[1, 1, 1]',
    '-1',
    1,
    true,
    now()
from selected_task
on conflict (task_id, order_no) do update
set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;

commit;
