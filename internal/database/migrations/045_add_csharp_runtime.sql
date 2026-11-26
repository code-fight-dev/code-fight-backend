-- Enable the validated plain-text starter on Mirror Log.
update tasks
set frontend_payload = jsonb_set(
        jsonb_set(frontend_payload, '{languageVersions,csharp}', '"12.0"'::jsonb),
        '{starterCodeByLanguage,csharp}', to_jsonb($csharp$
using System;

class Program
{
    static string Solve(string input)
    {
        return "";
    }

    static void Main()
    {
        Console.Write(Solve(Console.In.ReadToEnd()));
    }
}
$csharp$::text)),
    updated_at = clock_timestamp()
where slug = 'mirror-log' and frontend_payload->>'kind' = 'algorithmic';

insert into task_languages (id, task_id, language, language_version, is_enabled, created_at)
select gen_random_uuid(), id, 'csharp', '12.0', true, clock_timestamp()
from tasks where slug = 'mirror-log' and frontend_payload->>'kind' = 'algorithmic'
on conflict (task_id, language, language_version) do update set is_enabled = true;
