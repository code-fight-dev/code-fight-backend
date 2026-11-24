-- Align built-in C23 tasks with the Judge language registry.
-- Judge identifies C23 as language='c', version='23'.

insert into task_languages (
    id,
    task_id,
    language,
    language_version,
    is_enabled,
    created_at
)
select
    gen_random_uuid(),
    tl.task_id,
    'c',
    '23',
    true,
    now()
from task_languages tl
join tasks t on t.id = tl.task_id
where tl.language = 'c'
  and tl.language_version = 'c23'
  and tl.is_enabled = true
  and t.frontend_payload->>'kind' = 'algorithmic'
on conflict (task_id, language, language_version)
do update set is_enabled = true;

-- Disable the obsolete version for those tasks.
update task_languages tl
set is_enabled = false
from tasks t
where tl.task_id = t.id
  and tl.language = 'c'
  and tl.language_version = 'c23'
  and t.frontend_payload->>'kind' = 'algorithmic';

-- Update versions exposed to the frontend.
update tasks
set
    frontend_payload = jsonb_set(
        frontend_payload,
        '{languageVersions,c}',
        to_jsonb('23'::text),
        true
    ),
    updated_at = now()
where frontend_payload->>'kind' = 'algorithmic'
  and frontend_payload->'languageVersions'->>'c' = 'c23'
  and exists (
      select 1
      from task_languages tl
      where tl.task_id = tasks.id
        and tl.language = 'c'
        and tl.language_version = '23'
        and tl.is_enabled = true
  );