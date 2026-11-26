-- Mirror Log uses plain text input. Keep custom tasks and other challenges
-- on their existing language configurations until their Java templates are validated.
update tasks
set frontend_payload = jsonb_set(
        jsonb_set(frontend_payload, '{languageVersions,java}', '"21"'::jsonb),
        '{starterCodeByLanguage,java}', to_jsonb($java$
import java.nio.charset.StandardCharsets;

public class Main {
    static String solve(String input) {
        return "";
    }

    public static void main(String[] args) throws Exception {
        String input = new String(System.in.readAllBytes(), StandardCharsets.UTF_8);
        System.out.print(solve(input));
    }
}
$java$::text)),
    updated_at = clock_timestamp()
where slug = 'mirror-log' and frontend_payload->>'kind' = 'algorithmic';

insert into task_languages (id, task_id, language, language_version, is_enabled, created_at)
select gen_random_uuid(), id, 'java', '21', true, clock_timestamp()
from tasks where slug = 'mirror-log' and frontend_payload->>'kind' = 'algorithmic'
on conflict (task_id, language, language_version) do update set is_enabled = true;
