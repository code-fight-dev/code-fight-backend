begin;

-- signal-stabilizer
insert into tasks (
    id, slug, title, statement_md, difficulty, time_limit_ms, memory_limit_kb, is_active, frontend_payload, created_at, updated_at
) values (
    '2db3f0fb-9c15-4f58-9f59-84f74ef5f31f',
    'signal-stabilizer',
    'Signal Stabilizer',
    'A monitoring node receives a stream of signed sensor adjustments. Starting from zero, apply each adjustment in order and track the running total.

Return the zero-based index of the first adjustment after which the running total is inside the inclusive safe band. If the stream never stabilizes, return -1.',
    'easy',
    2000,
    262144,
    true,
    '{"id":"challenge-001","slug":"signal-stabilizer","title":"Signal Stabilizer","difficulty":"Easy","summary":"Find the first reading that keeps a running sensor total within a safe band.","description":["A monitoring node receives a stream of signed sensor adjustments. Starting from zero, apply each adjustment in order and track the running total.","Return the zero-based index of the first adjustment after which the running total is inside the inclusive safe band. If the stream never stabilizes, return -1."],"examples":[{"title":"Stabilizes after a dip","input":"band = [-2, 3], readings = [5, -7, 1, 2]","output":"2","explanation":"The running totals are 5, -2, -1, and 1. Index 2 is the first total inside the band after the out-of-band start."},{"title":"No safe reading","input":"band = [4, 6], readings = [1, 1, 1]","output":"-1"}],"constraints":["1 <= readings.length <= 100,000","-10,000 <= readings[i] <= 10,000","band[0] <= band[1]","The running total fits in a signed 64-bit integer."],"notes":["A single pass is enough if you update the running total incrementally."],"tags":["Array","Prefix Sum","Simulation"],"category":"Warmup","kind":"algorithmic","supportedLanguages":["typescript","javascript","python","java","cpp","go","rust","zig","csharp","c","ruby","php"],"starterCodeByLanguage":{"typescript":"// Signal Stabilizer\n// Parse stdin and print the answer.\nimport { readFileSync } from \"node:fs\";\n\nfunction solve(input: string): string {\n  return \"\";\n}\n\nconst input = readFileSync(0, \"utf8\");\nprocess.stdout.write(solve(input));\n","javascript":"// Signal Stabilizer\n// Parse stdin and print the answer.\nconst fs = require(\"node:fs\");\n\nfunction solve(input) {\n  return \"\";\n}\n\nconst input = fs.readFileSync(0, \"utf8\");\nprocess.stdout.write(solve(input));\n","python":"# Signal Stabilizer\n# Parse stdin and print the answer.\nimport sys\n\ndef solve(input_data: str) -> str:\n    return \"\"\n\ninput_data = sys.stdin.read()\nprint(solve(input_data), end=\"\")\n","java":"// Signal Stabilizer\n// Parse stdin and print the answer.\nimport java.io.*;\n\npublic class Main {\n    static String solve(String input) {\n        return \"\";\n    }\n\n    public static void main(String[] args) throws Exception {\n        String input = new String(System.in.readAllBytes());\n        System.out.print(solve(input));\n    }\n}\n","cpp":"// Signal Stabilizer\n// Parse stdin and print the answer.\n#include <bits/stdc++.h>\nusing namespace std;\n\nstring solve(const string& input) {\n    return \"\";\n}\n\nint main() {\n    ios::sync_with_stdio(false);\n    cin.tie(nullptr);\n\n    string input((istreambuf_iterator<char>(cin)), istreambuf_iterator<char>());\n    cout << solve(input);\n    return 0;\n}\n","go":"// Signal Stabilizer\n// Parse stdin and print the answer.\npackage main\n\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"os\"\n)\n\nfunc Solve(input string) string {\n\treturn \"\"\n}\n\nfunc main() {\n\tinput, err := io.ReadAll(os.Stdin)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Print(Solve(string(input)))\n}\n","rust":"// Signal Stabilizer\n// Parse stdin and print the answer.\nuse std::io::{self, Read};\n\nfn solve(input: &str) -> String {\n    String::new()\n}\n\nfn main() {\n    let mut input = String::new();\n    io::stdin().read_to_string(&mut input).unwrap();\n    print!(\"{}\", solve(&input));\n}\n","zig":"// Signal Stabilizer\n// Parse stdin and print the answer.\nconst std = @import(\"std\");\n\nfn solve(input: []const u8, allocator: std.mem.Allocator) ![]u8 {\n    _ = allocator;\n    _ = input;\n    return \"\";\n}\n\npub fn main() !void {\n    var gpa = std.heap.GeneralPurposeAllocator(.{}){};\n    defer _ = gpa.deinit();\n    const allocator = gpa.allocator();\n\n    var stdin = std.io.getStdIn().reader();\n    const input = try stdin.readAllAlloc(allocator, 1 << 20);\n    defer allocator.free(input);\n\n    const output = try solve(input, allocator);\n    defer allocator.free(output);\n\n    try std.io.getStdOut().writer().print(\"{s}\", .{output});\n}\n","csharp":"// Signal Stabilizer\n// Parse stdin and print the answer.\nusing System;\n\nclass Program\n{\n    static string Solve(string input)\n    {\n        return \"\";\n    }\n\n    static void Main()\n    {\n        var input = Console.In.ReadToEnd();\n        Console.Write(Solve(input));\n    }\n}\n","c":"// Signal Stabilizer\n// Parse stdin and print the answer.\n#include <stdio.h>\n#include <stdlib.h>\n\nchar* solve(const char* input) {\n    (void)input;\n    char* out = malloc(1);\n    out[0] = ''\\0'';\n    return out;\n}\n\nint main(void) {\n    char* input = NULL;\n    size_t cap = 0;\n    size_t len = 0;\n\n    int ch;\n    while ((ch = getchar()) != EOF) {\n        if (len + 1 >= cap) {\n            cap = cap == 0 ? 256 : cap * 2;\n            input = realloc(input, cap);\n        }\n        input[len++] = (char)ch;\n    }\n    if (!input) {\n        input = malloc(1);\n    }\n    input[len] = ''\\0'';\n\n    char* out = solve(input);\n    fputs(out, stdout);\n\n    free(out);\n    free(input);\n    return 0;\n}\n","ruby":"# Signal Stabilizer\n# Parse stdin and print the answer.\ndef solve(input)\n  \"\"\nend\n\ninput = STDIN.read\nprint solve(input)\n","php":"<?php\n// Signal Stabilizer\n// Parse stdin and print the answer.\nfunction solve(string $input): string {\n    return \"\";\n}\n\n$input = stream_get_contents(STDIN);\necho solve($input);\n"},"acceptanceRate":78,"estimatedMinutes":12,"attempts":18320,"popularity":94,"createdAt":"2026-03-21","progress":"solved","testCases":[{"name":"Case 1","input":"[-2, 3]\n[5, -7, 1, 2]","expectedOutput":"2"},{"name":"Case 2","input":"[4, 6]\n[1, 1, 1]","expectedOutput":"-1"}],"languageVersions":{"typescript":"5.8","javascript":"node22","python":"3.12","java":"21","cpp":"gnu++23","go":"1.24","rust":"1.80","zig":"0.13","csharp":"12.0","c":"c23","ruby":"3.3","php":"8.3"}}'::jsonb,
    '2026-03-21T00:00:00Z'::timestamptz,
    now()
)
on conflict (slug) do update set
    title = excluded.title,
    statement_md = excluded.statement_md,
    difficulty = excluded.difficulty,
    time_limit_ms = excluded.time_limit_ms,
    memory_limit_kb = excluded.memory_limit_kb,
    is_active = true,
    frontend_payload = excluded.frontend_payload,
    updated_at = now();
update task_languages
set is_enabled = false
where task_id = '2db3f0fb-9c15-4f58-9f59-84f74ef5f31f'::uuid
  and language not in ('typescript', 'javascript', 'python', 'java', 'cpp', 'go', 'rust', 'zig', 'csharp', 'c', 'ruby', 'php');
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '29aab40e-bf95-487a-8cd7-cfe631ed8eae',
    '2db3f0fb-9c15-4f58-9f59-84f74ef5f31f'::uuid,
    'typescript',
    '5.8',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'ba56e8df-bfe1-447c-83e6-c36dbed4218c',
    '2db3f0fb-9c15-4f58-9f59-84f74ef5f31f'::uuid,
    'javascript',
    'node22',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'e22fd47b-60cc-46bf-838c-fc10492feb0e',
    '2db3f0fb-9c15-4f58-9f59-84f74ef5f31f'::uuid,
    'python',
    '3.12',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '8143db38-5d4b-4496-b243-0819c8da6cfc',
    '2db3f0fb-9c15-4f58-9f59-84f74ef5f31f'::uuid,
    'java',
    '21',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'e788e700-645d-443f-81f7-a541ca7e8992',
    '2db3f0fb-9c15-4f58-9f59-84f74ef5f31f'::uuid,
    'cpp',
    'gnu++23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '49e22d66-60e0-4310-ac7b-fd60af8fb252',
    '2db3f0fb-9c15-4f58-9f59-84f74ef5f31f'::uuid,
    'go',
    '1.24',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'd5f167a4-072c-4bce-9220-96fe0d7759da',
    '2db3f0fb-9c15-4f58-9f59-84f74ef5f31f'::uuid,
    'rust',
    '1.80',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '1c1063fb-0b3c-4886-ae64-3bd2ce60623d',
    '2db3f0fb-9c15-4f58-9f59-84f74ef5f31f'::uuid,
    'zig',
    '0.13',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '3357e379-9b94-4785-b855-a4a66cb29460',
    '2db3f0fb-9c15-4f58-9f59-84f74ef5f31f'::uuid,
    'csharp',
    '12.0',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'eaec34d3-ca5f-484f-bd36-cb6d3d69e688',
    '2db3f0fb-9c15-4f58-9f59-84f74ef5f31f'::uuid,
    'c',
    'c23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '2ea8f990-ec71-49ff-83b8-03fb7f173acd',
    '2db3f0fb-9c15-4f58-9f59-84f74ef5f31f'::uuid,
    'ruby',
    '3.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'a1e43495-36d2-4349-800f-12eaf1b52d33',
    '2db3f0fb-9c15-4f58-9f59-84f74ef5f31f'::uuid,
    'php',
    '8.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    'e08f299d-87f7-4be0-89c6-7480f17e17ca',
    '2db3f0fb-9c15-4f58-9f59-84f74ef5f31f'::uuid,
    'sample',
    1,
    '[-2, 3]
[5, -7, 1, 2]',
    '2',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    'dcab00b4-a08a-4794-b701-29d794b30c19',
    '2db3f0fb-9c15-4f58-9f59-84f74ef5f31f'::uuid,
    'sample',
    2,
    '[4, 6]
[1, 1, 1]',
    '-1',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
update task_test_cases
set is_active = false
where task_id = '2db3f0fb-9c15-4f58-9f59-84f74ef5f31f'::uuid
  and kind = 'sample'
  and order_no not in (1, 2);

-- mirror-log
insert into tasks (
    id, slug, title, statement_md, difficulty, time_limit_ms, memory_limit_kb, is_active, frontend_payload, created_at, updated_at
) values (
    '5fe22ab4-de09-4677-b37e-ef78169b61fc',
    'mirror-log',
    'Mirror Log',
    'A terminal recorder stores commands with spaces, dashes, and timestamps mixed into the stream. For a quick integrity check, only lowercase letters and digits matter.

Return true if the meaningful characters form a palindrome. Character case should be ignored.',
    'easy',
    2000,
    262144,
    true,
    '{"id":"challenge-002","slug":"mirror-log","title":"Mirror Log","difficulty":"Easy","summary":"Check whether a command log reads the same after ignoring separators.","description":["A terminal recorder stores commands with spaces, dashes, and timestamps mixed into the stream. For a quick integrity check, only lowercase letters and digits matter.","Return true if the meaningful characters form a palindrome. Character case should be ignored."],"examples":[{"title":"Balanced log","input":"log = \"Deploy-101 -- yolped\"","output":"true","explanation":"The meaningful stream is deploy101yolped, which mirrors around 101."},{"title":"Mismatched log","input":"log = \"cache-evict-42\"","output":"false"}],"constraints":["1 <= log.length <= 200,000","The log may contain letters, digits, spaces, punctuation, and symbols.","Only ASCII alphanumeric characters participate in the check."],"notes":["Use two pointers so the extra memory stays constant."],"tags":["String","Two Pointers"],"category":"Strings","kind":"algorithmic","supportedLanguages":["typescript","javascript","python","java","cpp","go","rust","zig","csharp","c","ruby","php"],"starterCodeByLanguage":{"typescript":"// Mirror Log\n// Parse the provided input and return the answer as a string.\nexport function solve(input: string): string {\n  return \"\";\n}\n","javascript":"// Mirror Log\n// Parse the provided input and return the answer as a string.\nfunction solve(input) {\n  return \"\";\n}\n\nmodule.exports = { solve };\n","python":"# Mirror Log\n# Parse the provided input and return the answer as a string.\ndef solve(input_data: str) -> str:\n    return \"\"\n","java":"// Mirror Log\n// Parse the provided input and return the answer as a string.\nclass Solution {\n    public String solve(String input) {\n        return \"\";\n    }\n}\n","cpp":"// Mirror Log\n// Parse the provided input and return the answer as a string.\n#include <string>\nusing namespace std;\n\nclass Solution {\npublic:\n    string solve(const string& input) {\n        return \"\";\n    }\n};\n","go":"// Mirror Log\n// Parse stdin and print the answer.\npackage main\n\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"os\"\n)\n\nfunc Solve(input string) string {\n\treturn \"\"\n}\n\nfunc main() {\n\tinput, err := io.ReadAll(os.Stdin)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Print(Solve(string(input)))\n}\n","rust":"// Mirror Log\n// Parse the provided input and return the answer as a string.\npub fn solve(input: &str) -> String {\n    String::new()\n}\n","zig":"// Mirror Log\n// Parse stdin and print the answer.\nconst std = @import(\"std\");\n\nfn solve(input: []const u8, allocator: std.mem.Allocator) ![]u8 {\n    _ = allocator;\n    _ = input;\n    return \"\";\n}\n\npub fn main() !void {\n    var gpa = std.heap.GeneralPurposeAllocator(.{}){};\n    defer _ = gpa.deinit();\n    const allocator = gpa.allocator();\n\n    var stdin = std.io.getStdIn().reader();\n    const input = try stdin.readAllAlloc(allocator, 1 << 20);\n    defer allocator.free(input);\n\n    const output = try solve(input, allocator);\n    defer allocator.free(output);\n\n    try std.io.getStdOut().writer().print(\"{s}\", .{output});\n}\n","csharp":"// Mirror Log\n// Parse stdin and print the answer.\nusing System;\n\nclass Program\n{\n    static string Solve(string input)\n    {\n        return \"\";\n    }\n\n    static void Main()\n    {\n        var input = Console.In.ReadToEnd();\n        Console.Write(Solve(input));\n    }\n}\n","c":"// Mirror Log\n// Parse stdin and print the answer.\n#include <stdio.h>\n#include <stdlib.h>\n\nchar* solve(const char* input) {\n    (void)input;\n    char* out = malloc(1);\n    out[0] = ''\\0'';\n    return out;\n}\n\nint main(void) {\n    char* input = NULL;\n    size_t cap = 0;\n    size_t len = 0;\n\n    int ch;\n    while ((ch = getchar()) != EOF) {\n        if (len + 1 >= cap) {\n            cap = cap == 0 ? 256 : cap * 2;\n            input = realloc(input, cap);\n        }\n        input[len++] = (char)ch;\n    }\n    if (!input) {\n        input = malloc(1);\n    }\n    input[len] = ''\\0'';\n\n    char* out = solve(input);\n    fputs(out, stdout);\n\n    free(out);\n    free(input);\n    return 0;\n}\n","ruby":"# Mirror Log\n# Parse stdin and print the answer.\ndef solve(input)\n  \"\"\nend\n\ninput = STDIN.read\nprint solve(input)\n","php":"<?php\n// Mirror Log\n// Parse stdin and print the answer.\nfunction solve(string $input): string {\n    return \"\";\n}\n\n$input = stream_get_contents(STDIN);\necho solve($input);\n"},"acceptanceRate":83,"estimatedMinutes":10,"attempts":14240,"popularity":88,"createdAt":"2026-03-18","progress":"in-progress","testCases":[{"name":"Case 1","input":"Deploy-101 -- yolped","expectedOutput":"true"},{"name":"Case 2","input":"cache-evict-42","expectedOutput":"false"}],"languageVersions":{"typescript":"5.8","javascript":"node22","python":"3.12","java":"21","cpp":"gnu++23","go":"1.24","rust":"1.80","zig":"0.13","csharp":"12.0","c":"c23","ruby":"3.3","php":"8.3"}}'::jsonb,
    '2026-03-18T00:00:00Z'::timestamptz,
    now()
)
on conflict (slug) do update set
    title = excluded.title,
    statement_md = excluded.statement_md,
    difficulty = excluded.difficulty,
    time_limit_ms = excluded.time_limit_ms,
    memory_limit_kb = excluded.memory_limit_kb,
    is_active = true,
    frontend_payload = excluded.frontend_payload,
    updated_at = now();
update task_languages
set is_enabled = false
where task_id = '5fe22ab4-de09-4677-b37e-ef78169b61fc'::uuid
  and language not in ('typescript', 'javascript', 'python', 'java', 'cpp', 'go', 'rust', 'zig', 'csharp', 'c', 'ruby', 'php');
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '0d0f2f96-6a00-466a-9881-8e2d67b9debc',
    '5fe22ab4-de09-4677-b37e-ef78169b61fc'::uuid,
    'typescript',
    '5.8',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '234dfa13-cfa3-4ae4-a790-93790380404e',
    '5fe22ab4-de09-4677-b37e-ef78169b61fc'::uuid,
    'javascript',
    'node22',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'bbe68aea-a491-4abc-b9dc-b09932f1b850',
    '5fe22ab4-de09-4677-b37e-ef78169b61fc'::uuid,
    'python',
    '3.12',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'f8bba501-a74d-450c-8682-a489b021107e',
    '5fe22ab4-de09-4677-b37e-ef78169b61fc'::uuid,
    'java',
    '21',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'b371c8f1-c04b-4601-909c-d7d7905b6394',
    '5fe22ab4-de09-4677-b37e-ef78169b61fc'::uuid,
    'cpp',
    'gnu++23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '04dd9810-eda9-48c0-ae67-29c0bca77e16',
    '5fe22ab4-de09-4677-b37e-ef78169b61fc'::uuid,
    'go',
    '1.24',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '5a2d52ad-a957-472e-b661-9d22e1b4bcf3',
    '5fe22ab4-de09-4677-b37e-ef78169b61fc'::uuid,
    'rust',
    '1.80',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '82116882-88a5-4b21-bb9e-ca43c7fce6be',
    '5fe22ab4-de09-4677-b37e-ef78169b61fc'::uuid,
    'zig',
    '0.13',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '0d391dcf-5f73-4106-8781-ab7acf328717',
    '5fe22ab4-de09-4677-b37e-ef78169b61fc'::uuid,
    'csharp',
    '12.0',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'f036ea15-fe80-4af5-86c1-87dac2cc7c1c',
    '5fe22ab4-de09-4677-b37e-ef78169b61fc'::uuid,
    'c',
    'c23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '7e7e96f6-6d90-47a2-9399-503693753c89',
    '5fe22ab4-de09-4677-b37e-ef78169b61fc'::uuid,
    'ruby',
    '3.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'e357da6d-a8e1-4154-90ae-d644fb7b4856',
    '5fe22ab4-de09-4677-b37e-ef78169b61fc'::uuid,
    'php',
    '8.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    '7c1c4e52-4eb4-4ea2-96dc-11323391470e',
    '5fe22ab4-de09-4677-b37e-ef78169b61fc'::uuid,
    'sample',
    1,
    'Deploy-101 -- yolped',
    'true',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    'c86684cb-43af-4daa-b53b-5b0dc7eb5d7f',
    '5fe22ab4-de09-4677-b37e-ef78169b61fc'::uuid,
    'sample',
    2,
    'cache-evict-42',
    'false',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
update task_test_cases
set is_active = false
where task_id = '5fe22ab4-de09-4677-b37e-ef78169b61fc'::uuid
  and kind = 'sample'
  and order_no not in (1, 2);

-- cache-key-collisions
insert into tasks (
    id, slug, title, statement_md, difficulty, time_limit_ms, memory_limit_kb, is_active, frontend_payload, created_at, updated_at
) values (
    '4fd363ba-7aef-415d-9154-3a387759ed3f',
    'cache-key-collisions',
    'Cache Key Collisions',
    'A gateway normalizes cache keys before routing them. Normalization lowercases letters and removes underscores.

Given a list of keys, return every canonical form that appears more than once, along with the original keys that produced it. Preserve each group in the order its first collision is discovered.',
    'medium',
    3000,
    262144,
    true,
    '{"id":"challenge-003","slug":"cache-key-collisions","title":"Cache Key Collisions","difficulty":"Medium","summary":"Group API cache keys that normalize to the same canonical form.","description":["A gateway normalizes cache keys before routing them. Normalization lowercases letters and removes underscores.","Given a list of keys, return every canonical form that appears more than once, along with the original keys that produced it. Preserve each group in the order its first collision is discovered."],"examples":[{"title":"Two collision groups","input":"keys = [\"User_List\", \"userlist\", \"Order_Read\", \"orderread\", \"orders\"]","output":"[[\"userlist\", [\"User_List\", \"userlist\"]], [\"orderread\", [\"Order_Read\", \"orderread\"]]]"}],"constraints":["1 <= keys.length <= 50,000","1 <= keys[i].length <= 80","Keys contain letters, digits, underscores, and dashes.","The total input size is at most 1 MB."],"notes":["A map from normalized key to original keys gives stable grouping."],"tags":["Hash Table","String"],"category":"Data Structures","kind":"algorithmic","supportedLanguages":["typescript","javascript","python","java","cpp","go","rust","zig","csharp","c","ruby","php"],"starterCodeByLanguage":{"typescript":"// Cache Key Collisions\n// Parse the provided input and return the answer as a string.\nexport function solve(input: string): string {\n  return \"\";\n}\n","javascript":"// Cache Key Collisions\n// Parse the provided input and return the answer as a string.\nfunction solve(input) {\n  return \"\";\n}\n\nmodule.exports = { solve };\n","python":"# Cache Key Collisions\n# Parse the provided input and return the answer as a string.\ndef solve(input_data: str) -> str:\n    return \"\"\n","java":"// Cache Key Collisions\n// Parse the provided input and return the answer as a string.\nclass Solution {\n    public String solve(String input) {\n        return \"\";\n    }\n}\n","cpp":"// Cache Key Collisions\n// Parse the provided input and return the answer as a string.\n#include <string>\nusing namespace std;\n\nclass Solution {\npublic:\n    string solve(const string& input) {\n        return \"\";\n    }\n};\n","go":"// Cache Key Collisions\n// Parse stdin and print the answer.\npackage main\n\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"os\"\n)\n\nfunc Solve(input string) string {\n\treturn \"\"\n}\n\nfunc main() {\n\tinput, err := io.ReadAll(os.Stdin)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Print(Solve(string(input)))\n}\n","rust":"// Cache Key Collisions\n// Parse the provided input and return the answer as a string.\npub fn solve(input: &str) -> String {\n    String::new()\n}\n","zig":"// Cache Key Collisions\n// Parse stdin and print the answer.\nconst std = @import(\"std\");\n\nfn solve(input: []const u8, allocator: std.mem.Allocator) ![]u8 {\n    _ = allocator;\n    _ = input;\n    return \"\";\n}\n\npub fn main() !void {\n    var gpa = std.heap.GeneralPurposeAllocator(.{}){};\n    defer _ = gpa.deinit();\n    const allocator = gpa.allocator();\n\n    var stdin = std.io.getStdIn().reader();\n    const input = try stdin.readAllAlloc(allocator, 1 << 20);\n    defer allocator.free(input);\n\n    const output = try solve(input, allocator);\n    defer allocator.free(output);\n\n    try std.io.getStdOut().writer().print(\"{s}\", .{output});\n}\n","csharp":"// Cache Key Collisions\n// Parse stdin and print the answer.\nusing System;\n\nclass Program\n{\n    static string Solve(string input)\n    {\n        return \"\";\n    }\n\n    static void Main()\n    {\n        var input = Console.In.ReadToEnd();\n        Console.Write(Solve(input));\n    }\n}\n","c":"// Cache Key Collisions\n// Parse stdin and print the answer.\n#include <stdio.h>\n#include <stdlib.h>\n\nchar* solve(const char* input) {\n    (void)input;\n    char* out = malloc(1);\n    out[0] = ''\\0'';\n    return out;\n}\n\nint main(void) {\n    char* input = NULL;\n    size_t cap = 0;\n    size_t len = 0;\n\n    int ch;\n    while ((ch = getchar()) != EOF) {\n        if (len + 1 >= cap) {\n            cap = cap == 0 ? 256 : cap * 2;\n            input = realloc(input, cap);\n        }\n        input[len++] = (char)ch;\n    }\n    if (!input) {\n        input = malloc(1);\n    }\n    input[len] = ''\\0'';\n\n    char* out = solve(input);\n    fputs(out, stdout);\n\n    free(out);\n    free(input);\n    return 0;\n}\n","ruby":"# Cache Key Collisions\n# Parse stdin and print the answer.\ndef solve(input)\n  \"\"\nend\n\ninput = STDIN.read\nprint solve(input)\n","php":"<?php\n// Cache Key Collisions\n// Parse stdin and print the answer.\nfunction solve(string $input): string {\n    return \"\";\n}\n\n$input = stream_get_contents(STDIN);\necho solve($input);\n"},"acceptanceRate":61,"estimatedMinutes":24,"attempts":21980,"popularity":91,"createdAt":"2026-03-25","progress":"not-started","testCases":[{"name":"Case 1","input":"User_List,userlist,Order_Read,orderread,orders","expectedOutput":"userlist: User_List userlist\norderread: Order_Read orderread"}],"languageVersions":{"typescript":"5.8","javascript":"node22","python":"3.12","java":"21","cpp":"gnu++23","go":"1.24","rust":"1.80","zig":"0.13","csharp":"12.0","c":"c23","ruby":"3.3","php":"8.3"}}'::jsonb,
    '2026-03-25T00:00:00Z'::timestamptz,
    now()
)
on conflict (slug) do update set
    title = excluded.title,
    statement_md = excluded.statement_md,
    difficulty = excluded.difficulty,
    time_limit_ms = excluded.time_limit_ms,
    memory_limit_kb = excluded.memory_limit_kb,
    is_active = true,
    frontend_payload = excluded.frontend_payload,
    updated_at = now();
update task_languages
set is_enabled = false
where task_id = '4fd363ba-7aef-415d-9154-3a387759ed3f'::uuid
  and language not in ('typescript', 'javascript', 'python', 'java', 'cpp', 'go', 'rust', 'zig', 'csharp', 'c', 'ruby', 'php');
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'd004fe1f-119d-4f20-8450-fcc168552200',
    '4fd363ba-7aef-415d-9154-3a387759ed3f'::uuid,
    'typescript',
    '5.8',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'b5710834-7bcc-468a-8795-8fe965cbaef6',
    '4fd363ba-7aef-415d-9154-3a387759ed3f'::uuid,
    'javascript',
    'node22',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'b8f0c882-8a50-465f-8f95-fc39c2a1264b',
    '4fd363ba-7aef-415d-9154-3a387759ed3f'::uuid,
    'python',
    '3.12',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '05f2f114-e0a0-498a-a8a0-3e40b4562154',
    '4fd363ba-7aef-415d-9154-3a387759ed3f'::uuid,
    'java',
    '21',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '615a91a3-249f-4220-9a42-4261c5785120',
    '4fd363ba-7aef-415d-9154-3a387759ed3f'::uuid,
    'cpp',
    'gnu++23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '0674ca62-7af9-48ea-9aba-1740d1e85da4',
    '4fd363ba-7aef-415d-9154-3a387759ed3f'::uuid,
    'go',
    '1.24',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '3a7711f5-c71e-4079-8d8d-26721314c82a',
    '4fd363ba-7aef-415d-9154-3a387759ed3f'::uuid,
    'rust',
    '1.80',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'c3b8086a-3c2f-408c-ab34-a94fe9239a9a',
    '4fd363ba-7aef-415d-9154-3a387759ed3f'::uuid,
    'zig',
    '0.13',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '815cfec0-9361-430f-bfc7-6b3b193f8590',
    '4fd363ba-7aef-415d-9154-3a387759ed3f'::uuid,
    'csharp',
    '12.0',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '81ca3c04-5d47-4c6f-8e3b-3862a8395fab',
    '4fd363ba-7aef-415d-9154-3a387759ed3f'::uuid,
    'c',
    'c23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '4e7b1cf1-64ba-49d9-8a27-f826cccccb7f',
    '4fd363ba-7aef-415d-9154-3a387759ed3f'::uuid,
    'ruby',
    '3.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'bba2189c-da60-42ac-ae92-eb844737915a',
    '4fd363ba-7aef-415d-9154-3a387759ed3f'::uuid,
    'php',
    '8.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    '178f0bf9-93c5-4ded-ba89-14e47b49f0cc',
    '4fd363ba-7aef-415d-9154-3a387759ed3f'::uuid,
    'sample',
    1,
    'User_List,userlist,Order_Read,orderread,orders',
    'userlist: User_List userlist
orderread: Order_Read orderread',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
update task_test_cases
set is_active = false
where task_id = '4fd363ba-7aef-415d-9154-3a387759ed3f'::uuid
  and kind = 'sample'
  and order_no not in (1);

-- packet-window-median
insert into tasks (
    id, slug, title, statement_md, difficulty, time_limit_ms, memory_limit_kb, is_active, frontend_payload, created_at, updated_at
) values (
    'd393f703-9d79-49fa-a9a5-14a9eda1a2e1',
    'packet-window-median',
    'Packet Window Median',
    'A network probe records packet latency in milliseconds. For every contiguous window of size k, compute the median latency.

If k is even, the median is the average of the two middle values. Return medians in window order with one decimal place when needed.',
    'hard',
    5000,
    393216,
    true,
    '{"id":"challenge-004","slug":"packet-window-median","title":"Packet Window Median","difficulty":"Hard","summary":"Report the rolling median latency for every fixed-size packet window.","description":["A network probe records packet latency in milliseconds. For every contiguous window of size k, compute the median latency.","If k is even, the median is the average of the two middle values. Return medians in window order with one decimal place when needed."],"examples":[{"title":"Odd window","input":"latency = [7, 2, 9, 4, 1], k = 3","output":"[7, 4, 4]"},{"title":"Even window","input":"latency = [5, 1, 8, 2], k = 2","output":"[3, 4.5, 5]"}],"constraints":["1 <= k <= latency.length <= 100,000","0 <= latency[i] <= 1,000,000","Expected complexity is O(n log k)."],"notes":["A two-heap structure with lazy deletion keeps each window balanced."],"tags":["Heap","Sliding Window","Sorting"],"category":"Advanced","kind":"algorithmic","supportedLanguages":["typescript","javascript","python","java","cpp","go","rust","zig","csharp","c","ruby","php"],"starterCodeByLanguage":{"typescript":"// Packet Window Median\n// Parse the provided input and return the answer as a string.\nexport function solve(input: string): string {\n  return \"\";\n}\n","javascript":"// Packet Window Median\n// Parse the provided input and return the answer as a string.\nfunction solve(input) {\n  return \"\";\n}\n\nmodule.exports = { solve };\n","python":"# Packet Window Median\n# Parse the provided input and return the answer as a string.\ndef solve(input_data: str) -> str:\n    return \"\"\n","java":"// Packet Window Median\n// Parse the provided input and return the answer as a string.\nclass Solution {\n    public String solve(String input) {\n        return \"\";\n    }\n}\n","cpp":"// Packet Window Median\n// Parse the provided input and return the answer as a string.\n#include <string>\nusing namespace std;\n\nclass Solution {\npublic:\n    string solve(const string& input) {\n        return \"\";\n    }\n};\n","go":"// Packet Window Median\n// Parse stdin and print the answer.\npackage main\n\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"os\"\n)\n\nfunc Solve(input string) string {\n\treturn \"\"\n}\n\nfunc main() {\n\tinput, err := io.ReadAll(os.Stdin)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Print(Solve(string(input)))\n}\n","rust":"// Packet Window Median\n// Parse the provided input and return the answer as a string.\npub fn solve(input: &str) -> String {\n    String::new()\n}\n","zig":"// Packet Window Median\n// Parse stdin and print the answer.\nconst std = @import(\"std\");\n\nfn solve(input: []const u8, allocator: std.mem.Allocator) ![]u8 {\n    _ = allocator;\n    _ = input;\n    return \"\";\n}\n\npub fn main() !void {\n    var gpa = std.heap.GeneralPurposeAllocator(.{}){};\n    defer _ = gpa.deinit();\n    const allocator = gpa.allocator();\n\n    var stdin = std.io.getStdIn().reader();\n    const input = try stdin.readAllAlloc(allocator, 1 << 20);\n    defer allocator.free(input);\n\n    const output = try solve(input, allocator);\n    defer allocator.free(output);\n\n    try std.io.getStdOut().writer().print(\"{s}\", .{output});\n}\n","csharp":"// Packet Window Median\n// Parse stdin and print the answer.\nusing System;\n\nclass Program\n{\n    static string Solve(string input)\n    {\n        return \"\";\n    }\n\n    static void Main()\n    {\n        var input = Console.In.ReadToEnd();\n        Console.Write(Solve(input));\n    }\n}\n","c":"// Packet Window Median\n// Parse stdin and print the answer.\n#include <stdio.h>\n#include <stdlib.h>\n\nchar* solve(const char* input) {\n    (void)input;\n    char* out = malloc(1);\n    out[0] = ''\\0'';\n    return out;\n}\n\nint main(void) {\n    char* input = NULL;\n    size_t cap = 0;\n    size_t len = 0;\n\n    int ch;\n    while ((ch = getchar()) != EOF) {\n        if (len + 1 >= cap) {\n            cap = cap == 0 ? 256 : cap * 2;\n            input = realloc(input, cap);\n        }\n        input[len++] = (char)ch;\n    }\n    if (!input) {\n        input = malloc(1);\n    }\n    input[len] = ''\\0'';\n\n    char* out = solve(input);\n    fputs(out, stdout);\n\n    free(out);\n    free(input);\n    return 0;\n}\n","ruby":"# Packet Window Median\n# Parse stdin and print the answer.\ndef solve(input)\n  \"\"\nend\n\ninput = STDIN.read\nprint solve(input)\n","php":"<?php\n// Packet Window Median\n// Parse stdin and print the answer.\nfunction solve(string $input): string {\n    return \"\";\n}\n\n$input = stream_get_contents(STDIN);\necho solve($input);\n"},"acceptanceRate":37,"estimatedMinutes":45,"attempts":16870,"popularity":86,"createdAt":"2026-02-28","progress":"locked","testCases":[{"name":"Case 1","input":"[7, 2, 9, 4, 1]\n3","expectedOutput":"[7, 4, 4]"},{"name":"Case 2","input":"[5, 1, 8, 2]\n2","expectedOutput":"[3, 4.5, 5]"}],"languageVersions":{"typescript":"5.8","javascript":"node22","python":"3.12","java":"21","cpp":"gnu++23","go":"1.24","rust":"1.80","zig":"0.13","csharp":"12.0","c":"c23","ruby":"3.3","php":"8.3"}}'::jsonb,
    '2026-02-28T00:00:00Z'::timestamptz,
    now()
)
on conflict (slug) do update set
    title = excluded.title,
    statement_md = excluded.statement_md,
    difficulty = excluded.difficulty,
    time_limit_ms = excluded.time_limit_ms,
    memory_limit_kb = excluded.memory_limit_kb,
    is_active = true,
    frontend_payload = excluded.frontend_payload,
    updated_at = now();
update task_languages
set is_enabled = false
where task_id = 'd393f703-9d79-49fa-a9a5-14a9eda1a2e1'::uuid
  and language not in ('typescript', 'javascript', 'python', 'java', 'cpp', 'go', 'rust', 'zig', 'csharp', 'c', 'ruby', 'php');
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '6a540833-f46d-4641-952f-dfa380055544',
    'd393f703-9d79-49fa-a9a5-14a9eda1a2e1'::uuid,
    'typescript',
    '5.8',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'f89d6295-1bac-4e2e-a762-7cd8a82730fd',
    'd393f703-9d79-49fa-a9a5-14a9eda1a2e1'::uuid,
    'javascript',
    'node22',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'bf57fdfe-7830-4ef5-a609-f34f43107c5e',
    'd393f703-9d79-49fa-a9a5-14a9eda1a2e1'::uuid,
    'python',
    '3.12',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '6fd713cc-318d-4d8f-86fa-95b26d82d99c',
    'd393f703-9d79-49fa-a9a5-14a9eda1a2e1'::uuid,
    'java',
    '21',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '8d51dd3a-e830-4618-b61c-8c25aac2984f',
    'd393f703-9d79-49fa-a9a5-14a9eda1a2e1'::uuid,
    'cpp',
    'gnu++23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'caf7cf92-524f-468a-8060-b163069b5f33',
    'd393f703-9d79-49fa-a9a5-14a9eda1a2e1'::uuid,
    'go',
    '1.24',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'c35ce902-ae33-4734-a733-e3a9f9db589e',
    'd393f703-9d79-49fa-a9a5-14a9eda1a2e1'::uuid,
    'rust',
    '1.80',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '28205900-7757-46e8-8423-6c574591ad4b',
    'd393f703-9d79-49fa-a9a5-14a9eda1a2e1'::uuid,
    'zig',
    '0.13',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '6850fd22-0fa5-469c-ac97-cf8cbaea0272',
    'd393f703-9d79-49fa-a9a5-14a9eda1a2e1'::uuid,
    'csharp',
    '12.0',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '228a8ce5-757f-4b69-aba8-d86aa0c832b8',
    'd393f703-9d79-49fa-a9a5-14a9eda1a2e1'::uuid,
    'c',
    'c23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '5ccd346a-2742-4769-9992-e02f4c4ad529',
    'd393f703-9d79-49fa-a9a5-14a9eda1a2e1'::uuid,
    'ruby',
    '3.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '8c6b95c2-11c0-4ecf-b358-015e93d7dcbe',
    'd393f703-9d79-49fa-a9a5-14a9eda1a2e1'::uuid,
    'php',
    '8.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    '41a25e0d-6a58-47a0-a1e3-067af8f0849b',
    'd393f703-9d79-49fa-a9a5-14a9eda1a2e1'::uuid,
    'sample',
    1,
    '[7, 2, 9, 4, 1]
3',
    '[7, 4, 4]',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    '48fb05f5-cb14-41a1-9352-49c4e91ef487',
    'd393f703-9d79-49fa-a9a5-14a9eda1a2e1'::uuid,
    'sample',
    2,
    '[5, 1, 8, 2]
2',
    '[3, 4.5, 5]',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
update task_test_cases
set is_active = false
where task_id = 'd393f703-9d79-49fa-a9a5-14a9eda1a2e1'::uuid
  and kind = 'sample'
  and order_no not in (1, 2);

-- courier-routes
insert into tasks (
    id, slug, title, statement_md, difficulty, time_limit_ms, memory_limit_kb, is_active, frontend_payload, created_at, updated_at
) values (
    'c5f93653-dda3-442d-8ff0-8c477cdcf064',
    'courier-routes',
    'Courier Routes',
    'A delivery network is modeled as hubs connected by bidirectional handoff lanes. A courier can move across one lane per handoff.

Given the network, a start hub, and a target hub, return the minimum number of handoffs. If the target is unreachable, return -1.',
    'medium',
    3000,
    262144,
    true,
    '{"id":"challenge-005","slug":"courier-routes","title":"Courier Routes","difficulty":"Medium","summary":"Find the fewest handoffs needed to move a package through city hubs.","description":["A delivery network is modeled as hubs connected by bidirectional handoff lanes. A courier can move across one lane per handoff.","Given the network, a start hub, and a target hub, return the minimum number of handoffs. If the target is unreachable, return -1."],"examples":[{"title":"Reachable target","input":"hubs = 5, lanes = [[0,1],[1,2],[0,3],[3,4]], start = 0, target = 4","output":"2"}],"constraints":["1 <= hubs <= 100,000","0 <= lanes.length <= 200,000","Hubs are numbered from 0 to hubs - 1.","The graph may be disconnected."],"notes":["Breadth-first search is the natural fit for unweighted shortest paths."],"tags":["Graph","Queue"],"category":"Graphs","kind":"algorithmic","supportedLanguages":["typescript","javascript","python","java","cpp","go","rust","zig","csharp","c","ruby","php"],"starterCodeByLanguage":{"typescript":"// Courier Routes\n// Parse the provided input and return the answer as a string.\nexport function solve(input: string): string {\n  return \"\";\n}\n","javascript":"// Courier Routes\n// Parse the provided input and return the answer as a string.\nfunction solve(input) {\n  return \"\";\n}\n\nmodule.exports = { solve };\n","python":"# Courier Routes\n# Parse the provided input and return the answer as a string.\ndef solve(input_data: str) -> str:\n    return \"\"\n","java":"// Courier Routes\n// Parse the provided input and return the answer as a string.\nclass Solution {\n    public String solve(String input) {\n        return \"\";\n    }\n}\n","cpp":"// Courier Routes\n// Parse the provided input and return the answer as a string.\n#include <string>\nusing namespace std;\n\nclass Solution {\npublic:\n    string solve(const string& input) {\n        return \"\";\n    }\n};\n","go":"// Courier Routes\n// Parse stdin and print the answer.\npackage main\n\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"os\"\n)\n\nfunc Solve(input string) string {\n\treturn \"\"\n}\n\nfunc main() {\n\tinput, err := io.ReadAll(os.Stdin)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Print(Solve(string(input)))\n}\n","rust":"// Courier Routes\n// Parse the provided input and return the answer as a string.\npub fn solve(input: &str) -> String {\n    String::new()\n}\n","zig":"// Courier Routes\n// Parse stdin and print the answer.\nconst std = @import(\"std\");\n\nfn solve(input: []const u8, allocator: std.mem.Allocator) ![]u8 {\n    _ = allocator;\n    _ = input;\n    return \"\";\n}\n\npub fn main() !void {\n    var gpa = std.heap.GeneralPurposeAllocator(.{}){};\n    defer _ = gpa.deinit();\n    const allocator = gpa.allocator();\n\n    var stdin = std.io.getStdIn().reader();\n    const input = try stdin.readAllAlloc(allocator, 1 << 20);\n    defer allocator.free(input);\n\n    const output = try solve(input, allocator);\n    defer allocator.free(output);\n\n    try std.io.getStdOut().writer().print(\"{s}\", .{output});\n}\n","csharp":"// Courier Routes\n// Parse stdin and print the answer.\nusing System;\n\nclass Program\n{\n    static string Solve(string input)\n    {\n        return \"\";\n    }\n\n    static void Main()\n    {\n        var input = Console.In.ReadToEnd();\n        Console.Write(Solve(input));\n    }\n}\n","c":"// Courier Routes\n// Parse stdin and print the answer.\n#include <stdio.h>\n#include <stdlib.h>\n\nchar* solve(const char* input) {\n    (void)input;\n    char* out = malloc(1);\n    out[0] = ''\\0'';\n    return out;\n}\n\nint main(void) {\n    char* input = NULL;\n    size_t cap = 0;\n    size_t len = 0;\n\n    int ch;\n    while ((ch = getchar()) != EOF) {\n        if (len + 1 >= cap) {\n            cap = cap == 0 ? 256 : cap * 2;\n            input = realloc(input, cap);\n        }\n        input[len++] = (char)ch;\n    }\n    if (!input) {\n        input = malloc(1);\n    }\n    input[len] = ''\\0'';\n\n    char* out = solve(input);\n    fputs(out, stdout);\n\n    free(out);\n    free(input);\n    return 0;\n}\n","ruby":"# Courier Routes\n# Parse stdin and print the answer.\ndef solve(input)\n  \"\"\nend\n\ninput = STDIN.read\nprint solve(input)\n","php":"<?php\n// Courier Routes\n// Parse stdin and print the answer.\nfunction solve(string $input): string {\n    return \"\";\n}\n\n$input = stream_get_contents(STDIN);\necho solve($input);\n"},"acceptanceRate":64,"estimatedMinutes":26,"attempts":19510,"popularity":89,"createdAt":"2026-03-08","progress":"not-started","testCases":[{"name":"Case 1","input":"5\n0 1\n1 2\n0 3\n3 4\n0\n4","expectedOutput":"2"}],"languageVersions":{"typescript":"5.8","javascript":"node22","python":"3.12","java":"21","cpp":"gnu++23","go":"1.24","rust":"1.80","zig":"0.13","csharp":"12.0","c":"c23","ruby":"3.3","php":"8.3"}}'::jsonb,
    '2026-03-08T00:00:00Z'::timestamptz,
    now()
)
on conflict (slug) do update set
    title = excluded.title,
    statement_md = excluded.statement_md,
    difficulty = excluded.difficulty,
    time_limit_ms = excluded.time_limit_ms,
    memory_limit_kb = excluded.memory_limit_kb,
    is_active = true,
    frontend_payload = excluded.frontend_payload,
    updated_at = now();
update task_languages
set is_enabled = false
where task_id = 'c5f93653-dda3-442d-8ff0-8c477cdcf064'::uuid
  and language not in ('typescript', 'javascript', 'python', 'java', 'cpp', 'go', 'rust', 'zig', 'csharp', 'c', 'ruby', 'php');
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'dfaa8902-9e2f-439a-95a9-11e0527d5ab0',
    'c5f93653-dda3-442d-8ff0-8c477cdcf064'::uuid,
    'typescript',
    '5.8',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'c3a31a66-057d-4de4-859f-dc9051477995',
    'c5f93653-dda3-442d-8ff0-8c477cdcf064'::uuid,
    'javascript',
    'node22',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'c80ab6d0-5902-425f-9c0d-45d4dcc3377e',
    'c5f93653-dda3-442d-8ff0-8c477cdcf064'::uuid,
    'python',
    '3.12',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '0748c11b-a5bd-463d-8d9b-ad06b772b6dc',
    'c5f93653-dda3-442d-8ff0-8c477cdcf064'::uuid,
    'java',
    '21',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'b4440b25-efff-4185-b549-9af204ad44a4',
    'c5f93653-dda3-442d-8ff0-8c477cdcf064'::uuid,
    'cpp',
    'gnu++23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '2017bd08-53e4-419e-a50b-0e189aa2b222',
    'c5f93653-dda3-442d-8ff0-8c477cdcf064'::uuid,
    'go',
    '1.24',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'b6f4d803-a56a-44b6-a05f-98fe3ccd2d91',
    'c5f93653-dda3-442d-8ff0-8c477cdcf064'::uuid,
    'rust',
    '1.80',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'a4e9fa4e-1248-44eb-8c6f-5a0715ed3c2a',
    'c5f93653-dda3-442d-8ff0-8c477cdcf064'::uuid,
    'zig',
    '0.13',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'f7fbe7c1-7688-49c2-a474-445599f647f8',
    'c5f93653-dda3-442d-8ff0-8c477cdcf064'::uuid,
    'csharp',
    '12.0',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'a7542b86-6a94-426c-9606-c8db9cdba7cf',
    'c5f93653-dda3-442d-8ff0-8c477cdcf064'::uuid,
    'c',
    'c23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '047d1528-efda-476f-aba3-c44ef13c6856',
    'c5f93653-dda3-442d-8ff0-8c477cdcf064'::uuid,
    'ruby',
    '3.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '7dd20ad7-fdb1-470d-be6d-b63056d4f1c5',
    'c5f93653-dda3-442d-8ff0-8c477cdcf064'::uuid,
    'php',
    '8.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    '2d47003a-cc2c-467c-91d1-4b99067bd72b',
    'c5f93653-dda3-442d-8ff0-8c477cdcf064'::uuid,
    'sample',
    1,
    '5
0 1
1 2
0 3
3 4
0
4',
    '2',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
update task_test_cases
set is_active = false
where task_id = 'c5f93653-dda3-442d-8ff0-8c477cdcf064'::uuid
  and kind = 'sample'
  and order_no not in (1);

-- build-pipeline-order
insert into tasks (
    id, slug, title, statement_md, difficulty, time_limit_ms, memory_limit_kb, is_active, frontend_payload, created_at, updated_at
) values (
    '457309c0-0c34-472e-aa9e-b15971d788d3',
    'build-pipeline-order',
    'Build Pipeline Order',
    'A build farm receives jobs and dependency pairs. A pair [a, b] means job a must finish before job b can start.

Return any valid job order. If the dependencies contain a cycle, return an empty list.',
    'medium',
    3000,
    262144,
    true,
    '{"id":"challenge-006","slug":"build-pipeline-order","title":"Build Pipeline Order","difficulty":"Medium","summary":"Schedule build jobs when some jobs depend on others finishing first.","description":["A build farm receives jobs and dependency pairs. A pair [a, b] means job a must finish before job b can start.","Return any valid job order. If the dependencies contain a cycle, return an empty list."],"examples":[{"title":"A valid pipeline","input":"jobs = 4, dependencies = [[0,1],[0,2],[2,3]]","output":"[0, 1, 2, 3]","explanation":"Job 1 and job 2 can swap positions, as long as 0 appears before both and 2 appears before 3."}],"constraints":["1 <= jobs <= 80,000","0 <= dependencies.length <= 160,000","Dependencies may include duplicate pairs."],"notes":["Indegree tracking with a queue is a direct way to detect cycles."],"tags":["Graph","DFS","Queue"],"category":"Graphs","kind":"algorithmic","supportedLanguages":["typescript","javascript","python","java","cpp","go","rust","zig","csharp","c","ruby","php"],"starterCodeByLanguage":{"typescript":"// Build Pipeline Order\n// Parse the provided input and return the answer as a string.\nexport function solve(input: string): string {\n  return \"\";\n}\n","javascript":"// Build Pipeline Order\n// Parse the provided input and return the answer as a string.\nfunction solve(input) {\n  return \"\";\n}\n\nmodule.exports = { solve };\n","python":"# Build Pipeline Order\n# Parse the provided input and return the answer as a string.\ndef solve(input_data: str) -> str:\n    return \"\"\n","java":"// Build Pipeline Order\n// Parse the provided input and return the answer as a string.\nclass Solution {\n    public String solve(String input) {\n        return \"\";\n    }\n}\n","cpp":"// Build Pipeline Order\n// Parse the provided input and return the answer as a string.\n#include <string>\nusing namespace std;\n\nclass Solution {\npublic:\n    string solve(const string& input) {\n        return \"\";\n    }\n};\n","go":"// Build Pipeline Order\n// Parse stdin and print the answer.\npackage main\n\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"os\"\n)\n\nfunc Solve(input string) string {\n\treturn \"\"\n}\n\nfunc main() {\n\tinput, err := io.ReadAll(os.Stdin)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Print(Solve(string(input)))\n}\n","rust":"// Build Pipeline Order\n// Parse the provided input and return the answer as a string.\npub fn solve(input: &str) -> String {\n    String::new()\n}\n","zig":"// Build Pipeline Order\n// Parse stdin and print the answer.\nconst std = @import(\"std\");\n\nfn solve(input: []const u8, allocator: std.mem.Allocator) ![]u8 {\n    _ = allocator;\n    _ = input;\n    return \"\";\n}\n\npub fn main() !void {\n    var gpa = std.heap.GeneralPurposeAllocator(.{}){};\n    defer _ = gpa.deinit();\n    const allocator = gpa.allocator();\n\n    var stdin = std.io.getStdIn().reader();\n    const input = try stdin.readAllAlloc(allocator, 1 << 20);\n    defer allocator.free(input);\n\n    const output = try solve(input, allocator);\n    defer allocator.free(output);\n\n    try std.io.getStdOut().writer().print(\"{s}\", .{output});\n}\n","csharp":"// Build Pipeline Order\n// Parse stdin and print the answer.\nusing System;\n\nclass Program\n{\n    static string Solve(string input)\n    {\n        return \"\";\n    }\n\n    static void Main()\n    {\n        var input = Console.In.ReadToEnd();\n        Console.Write(Solve(input));\n    }\n}\n","c":"// Build Pipeline Order\n// Parse stdin and print the answer.\n#include <stdio.h>\n#include <stdlib.h>\n\nchar* solve(const char* input) {\n    (void)input;\n    char* out = malloc(1);\n    out[0] = ''\\0'';\n    return out;\n}\n\nint main(void) {\n    char* input = NULL;\n    size_t cap = 0;\n    size_t len = 0;\n\n    int ch;\n    while ((ch = getchar()) != EOF) {\n        if (len + 1 >= cap) {\n            cap = cap == 0 ? 256 : cap * 2;\n            input = realloc(input, cap);\n        }\n        input[len++] = (char)ch;\n    }\n    if (!input) {\n        input = malloc(1);\n    }\n    input[len] = ''\\0'';\n\n    char* out = solve(input);\n    fputs(out, stdout);\n\n    free(out);\n    free(input);\n    return 0;\n}\n","ruby":"# Build Pipeline Order\n# Parse stdin and print the answer.\ndef solve(input)\n  \"\"\nend\n\ninput = STDIN.read\nprint solve(input)\n","php":"<?php\n// Build Pipeline Order\n// Parse stdin and print the answer.\nfunction solve(string $input): string {\n    return \"\";\n}\n\n$input = stream_get_contents(STDIN);\necho solve($input);\n"},"acceptanceRate":55,"estimatedMinutes":30,"attempts":12770,"popularity":82,"createdAt":"2026-03-16","progress":"not-started","testCases":[{"name":"Case 1","input":"4\n0 1\n0 2\n2 3","expectedOutput":"[0, 1, 2, 3]"}],"languageVersions":{"typescript":"5.8","javascript":"node22","python":"3.12","java":"21","cpp":"gnu++23","go":"1.24","rust":"1.80","zig":"0.13","csharp":"12.0","c":"c23","ruby":"3.3","php":"8.3"}}'::jsonb,
    '2026-03-16T00:00:00Z'::timestamptz,
    now()
)
on conflict (slug) do update set
    title = excluded.title,
    statement_md = excluded.statement_md,
    difficulty = excluded.difficulty,
    time_limit_ms = excluded.time_limit_ms,
    memory_limit_kb = excluded.memory_limit_kb,
    is_active = true,
    frontend_payload = excluded.frontend_payload,
    updated_at = now();
update task_languages
set is_enabled = false
where task_id = '457309c0-0c34-472e-aa9e-b15971d788d3'::uuid
  and language not in ('typescript', 'javascript', 'python', 'java', 'cpp', 'go', 'rust', 'zig', 'csharp', 'c', 'ruby', 'php');
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '1dc7e836-e35c-46c8-b5c8-0879ebd4732f',
    '457309c0-0c34-472e-aa9e-b15971d788d3'::uuid,
    'typescript',
    '5.8',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '4f423523-8caa-44ec-8084-cf6dc7dcabd2',
    '457309c0-0c34-472e-aa9e-b15971d788d3'::uuid,
    'javascript',
    'node22',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'e9545ebd-0431-4a41-a021-7e90002b4828',
    '457309c0-0c34-472e-aa9e-b15971d788d3'::uuid,
    'python',
    '3.12',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'd05122ec-b2b6-4e54-b113-b2feaff5939d',
    '457309c0-0c34-472e-aa9e-b15971d788d3'::uuid,
    'java',
    '21',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '8e52e107-d596-4609-acab-42ed5faf8c83',
    '457309c0-0c34-472e-aa9e-b15971d788d3'::uuid,
    'cpp',
    'gnu++23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'e5c16de1-9787-4afd-ae6d-65a32948ff1e',
    '457309c0-0c34-472e-aa9e-b15971d788d3'::uuid,
    'go',
    '1.24',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'b9423d94-6300-4aeb-a2c9-41a98ac141d4',
    '457309c0-0c34-472e-aa9e-b15971d788d3'::uuid,
    'rust',
    '1.80',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '4ccf0778-950d-44eb-8dcf-8d4e93f8dd29',
    '457309c0-0c34-472e-aa9e-b15971d788d3'::uuid,
    'zig',
    '0.13',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'b3a304d4-afb6-4265-b45f-686c61172448',
    '457309c0-0c34-472e-aa9e-b15971d788d3'::uuid,
    'csharp',
    '12.0',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '25bc0172-e693-4c74-8902-40f85177241c',
    '457309c0-0c34-472e-aa9e-b15971d788d3'::uuid,
    'c',
    'c23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '79e353a9-29ea-4302-ab3a-ca8fa4e973dc',
    '457309c0-0c34-472e-aa9e-b15971d788d3'::uuid,
    'ruby',
    '3.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '8bb414f0-ad4a-4723-90fa-efbd4ee1f6cb',
    '457309c0-0c34-472e-aa9e-b15971d788d3'::uuid,
    'php',
    '8.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    'f3c72d15-ad0a-460a-a5c8-c48fc2092e5b',
    '457309c0-0c34-472e-aa9e-b15971d788d3'::uuid,
    'sample',
    1,
    '4
0 1
0 2
2 3',
    '[0, 1, 2, 3]',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
update task_test_cases
set is_active = false
where task_id = '457309c0-0c34-472e-aa9e-b15971d788d3'::uuid
  and kind = 'sample'
  and order_no not in (1);

-- adaptive-pricing
insert into tasks (
    id, slug, title, statement_md, difficulty, time_limit_ms, memory_limit_kb, is_active, frontend_payload, created_at, updated_at
) values (
    '7096f399-05eb-4766-bcd9-1c6599022ab4',
    'adaptive-pricing',
    'Adaptive Pricing',
    'A store tracks a projected price for the same item over several days. You may discount inventory on one day and restore the price on a later day.

Return the largest possible positive price gain. If no later day improves the price, return 0.',
    'easy',
    2000,
    262144,
    true,
    '{"id":"challenge-007","slug":"adaptive-pricing","title":"Adaptive Pricing","difficulty":"Easy","summary":"Choose the best single discount day and recovery day for a product.","description":["A store tracks a projected price for the same item over several days. You may discount inventory on one day and restore the price on a later day.","Return the largest possible positive price gain. If no later day improves the price, return 0."],"examples":[{"title":"Profitable recovery","input":"prices = [12, 7, 10, 5, 13]","output":"8"},{"title":"No recovery","input":"prices = [9, 8, 6]","output":"0"}],"constraints":["1 <= prices.length <= 100,000","0 <= prices[i] <= 1,000,000","The discount day must occur before the recovery day."],"notes":["Track the lowest price seen so far while scanning left to right."],"tags":["Array","Math","Greedy"],"category":"Warmup","kind":"algorithmic","supportedLanguages":["typescript","javascript","python","java","cpp","go","rust","zig","csharp","c","ruby","php"],"starterCodeByLanguage":{"typescript":"// Adaptive Pricing\n// Parse the provided input and return the answer as a string.\nexport function solve(input: string): string {\n  return \"\";\n}\n","javascript":"// Adaptive Pricing\n// Parse the provided input and return the answer as a string.\nfunction solve(input) {\n  return \"\";\n}\n\nmodule.exports = { solve };\n","python":"# Adaptive Pricing\n# Parse the provided input and return the answer as a string.\ndef solve(input_data: str) -> str:\n    return \"\"\n","java":"// Adaptive Pricing\n// Parse the provided input and return the answer as a string.\nclass Solution {\n    public String solve(String input) {\n        return \"\";\n    }\n}\n","cpp":"// Adaptive Pricing\n// Parse the provided input and return the answer as a string.\n#include <string>\nusing namespace std;\n\nclass Solution {\npublic:\n    string solve(const string& input) {\n        return \"\";\n    }\n};\n","go":"// Adaptive Pricing\n// Parse stdin and print the answer.\npackage main\n\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"os\"\n)\n\nfunc Solve(input string) string {\n\treturn \"\"\n}\n\nfunc main() {\n\tinput, err := io.ReadAll(os.Stdin)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Print(Solve(string(input)))\n}\n","rust":"// Adaptive Pricing\n// Parse the provided input and return the answer as a string.\npub fn solve(input: &str) -> String {\n    String::new()\n}\n","zig":"// Adaptive Pricing\n// Parse stdin and print the answer.\nconst std = @import(\"std\");\n\nfn solve(input: []const u8, allocator: std.mem.Allocator) ![]u8 {\n    _ = allocator;\n    _ = input;\n    return \"\";\n}\n\npub fn main() !void {\n    var gpa = std.heap.GeneralPurposeAllocator(.{}){};\n    defer _ = gpa.deinit();\n    const allocator = gpa.allocator();\n\n    var stdin = std.io.getStdIn().reader();\n    const input = try stdin.readAllAlloc(allocator, 1 << 20);\n    defer allocator.free(input);\n\n    const output = try solve(input, allocator);\n    defer allocator.free(output);\n\n    try std.io.getStdOut().writer().print(\"{s}\", .{output});\n}\n","csharp":"// Adaptive Pricing\n// Parse stdin and print the answer.\nusing System;\n\nclass Program\n{\n    static string Solve(string input)\n    {\n        return \"\";\n    }\n\n    static void Main()\n    {\n        var input = Console.In.ReadToEnd();\n        Console.Write(Solve(input));\n    }\n}\n","c":"// Adaptive Pricing\n// Parse stdin and print the answer.\n#include <stdio.h>\n#include <stdlib.h>\n\nchar* solve(const char* input) {\n    (void)input;\n    char* out = malloc(1);\n    out[0] = ''\\0'';\n    return out;\n}\n\nint main(void) {\n    char* input = NULL;\n    size_t cap = 0;\n    size_t len = 0;\n\n    int ch;\n    while ((ch = getchar()) != EOF) {\n        if (len + 1 >= cap) {\n            cap = cap == 0 ? 256 : cap * 2;\n            input = realloc(input, cap);\n        }\n        input[len++] = (char)ch;\n    }\n    if (!input) {\n        input = malloc(1);\n    }\n    input[len] = ''\\0'';\n\n    char* out = solve(input);\n    fputs(out, stdout);\n\n    free(out);\n    free(input);\n    return 0;\n}\n","ruby":"# Adaptive Pricing\n# Parse stdin and print the answer.\ndef solve(input)\n  \"\"\nend\n\ninput = STDIN.read\nprint solve(input)\n","php":"<?php\n// Adaptive Pricing\n// Parse stdin and print the answer.\nfunction solve(string $input): string {\n    return \"\";\n}\n\n$input = stream_get_contents(STDIN);\necho solve($input);\n"},"acceptanceRate":81,"estimatedMinutes":14,"attempts":26840,"popularity":96,"createdAt":"2026-02-24","progress":"solved","testCases":[{"name":"Case 1","input":"[12, 7, 10, 5, 13]","expectedOutput":"8"},{"name":"Case 2","input":"[9, 8, 6]","expectedOutput":"0"}],"languageVersions":{"typescript":"5.8","javascript":"node22","python":"3.12","java":"21","cpp":"gnu++23","go":"1.24","rust":"1.80","zig":"0.13","csharp":"12.0","c":"c23","ruby":"3.3","php":"8.3"}}'::jsonb,
    '2026-02-24T00:00:00Z'::timestamptz,
    now()
)
on conflict (slug) do update set
    title = excluded.title,
    statement_md = excluded.statement_md,
    difficulty = excluded.difficulty,
    time_limit_ms = excluded.time_limit_ms,
    memory_limit_kb = excluded.memory_limit_kb,
    is_active = true,
    frontend_payload = excluded.frontend_payload,
    updated_at = now();
update task_languages
set is_enabled = false
where task_id = '7096f399-05eb-4766-bcd9-1c6599022ab4'::uuid
  and language not in ('typescript', 'javascript', 'python', 'java', 'cpp', 'go', 'rust', 'zig', 'csharp', 'c', 'ruby', 'php');
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'a7e7b07c-be9b-4af0-866a-81b09e13acff',
    '7096f399-05eb-4766-bcd9-1c6599022ab4'::uuid,
    'typescript',
    '5.8',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '0c4f5e93-0640-44a0-ae57-6ed9d5b78c98',
    '7096f399-05eb-4766-bcd9-1c6599022ab4'::uuid,
    'javascript',
    'node22',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'f1ef136d-4c12-4a10-acf3-8066a94e9df9',
    '7096f399-05eb-4766-bcd9-1c6599022ab4'::uuid,
    'python',
    '3.12',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '14591e55-a37c-4370-9ba6-9c4b330bc9fe',
    '7096f399-05eb-4766-bcd9-1c6599022ab4'::uuid,
    'java',
    '21',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'f4511d43-2688-4fe3-866e-976348b0d4e5',
    '7096f399-05eb-4766-bcd9-1c6599022ab4'::uuid,
    'cpp',
    'gnu++23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '130eb382-f3af-4f96-bb37-0222f1589cd1',
    '7096f399-05eb-4766-bcd9-1c6599022ab4'::uuid,
    'go',
    '1.24',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '1a02179b-3207-4e19-9bf6-07d62315fa29',
    '7096f399-05eb-4766-bcd9-1c6599022ab4'::uuid,
    'rust',
    '1.80',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'f1a31ad8-40e6-489a-bab7-14254cf763cf',
    '7096f399-05eb-4766-bcd9-1c6599022ab4'::uuid,
    'zig',
    '0.13',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'dfa8c0a3-3da4-4053-a9d4-ed3dbe924de7',
    '7096f399-05eb-4766-bcd9-1c6599022ab4'::uuid,
    'csharp',
    '12.0',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '2a47a693-8284-4043-a437-7820e5121700',
    '7096f399-05eb-4766-bcd9-1c6599022ab4'::uuid,
    'c',
    'c23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'babbc3c0-d1f6-444e-b4dd-11829d1e7ae9',
    '7096f399-05eb-4766-bcd9-1c6599022ab4'::uuid,
    'ruby',
    '3.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '1e417c6e-ba8d-4c34-a337-99c298b5df20',
    '7096f399-05eb-4766-bcd9-1c6599022ab4'::uuid,
    'php',
    '8.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    '0f5c8b3f-5162-4eb4-aece-5c5f52078442',
    '7096f399-05eb-4766-bcd9-1c6599022ab4'::uuid,
    'sample',
    1,
    '[12, 7, 10, 5, 13]',
    '8',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    'e5483d63-186e-48b4-8684-c36432be5930',
    '7096f399-05eb-4766-bcd9-1c6599022ab4'::uuid,
    'sample',
    2,
    '[9, 8, 6]',
    '0',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
update task_test_cases
set is_active = false
where task_id = '7096f399-05eb-4766-bcd9-1c6599022ab4'::uuid
  and kind = 'sample'
  and order_no not in (1, 2);

-- binary-signal-boundary
insert into tasks (
    id, slug, title, statement_md, difficulty, time_limit_ms, memory_limit_kb, is_active, frontend_payload, created_at, updated_at
) values (
    'c88a7952-5f98-481a-a2f1-30fd7d49150b',
    'binary-signal-boundary',
    'Binary Signal Boundary',
    'A hardware probe emits a nondecreasing list of signal strengths. You need to find the first reading that is greater than or equal to a threshold.

Return the index of that reading, or the length of the list if every reading is below the threshold.',
    'medium',
    3000,
    262144,
    true,
    '{"id":"challenge-008","slug":"binary-signal-boundary","title":"Binary Signal Boundary","difficulty":"Medium","summary":"Locate the first index where a sorted signal crosses a threshold.","description":["A hardware probe emits a nondecreasing list of signal strengths. You need to find the first reading that is greater than or equal to a threshold.","Return the index of that reading, or the length of the list if every reading is below the threshold."],"examples":[{"title":"Threshold found","input":"signals = [1, 2, 2, 5, 9], threshold = 5","output":"3"},{"title":"Insertion at end","input":"signals = [1, 3, 4], threshold = 7","output":"3"}],"constraints":["0 <= signals.length <= 1,000,000","-1,000,000 <= signals[i], threshold <= 1,000,000","signals is sorted in nondecreasing order."],"notes":["Be precise with the open and closed sides of the search interval."],"tags":["Array","Binary Search"],"category":"Search","kind":"algorithmic","supportedLanguages":["typescript","javascript","python","java","cpp","go","rust","zig","csharp","c","ruby","php"],"starterCodeByLanguage":{"typescript":"// Binary Signal Boundary\n// Parse the provided input and return the answer as a string.\nexport function solve(input: string): string {\n  return \"\";\n}\n","javascript":"// Binary Signal Boundary\n// Parse the provided input and return the answer as a string.\nfunction solve(input) {\n  return \"\";\n}\n\nmodule.exports = { solve };\n","python":"# Binary Signal Boundary\n# Parse the provided input and return the answer as a string.\ndef solve(input_data: str) -> str:\n    return \"\"\n","java":"// Binary Signal Boundary\n// Parse the provided input and return the answer as a string.\nclass Solution {\n    public String solve(String input) {\n        return \"\";\n    }\n}\n","cpp":"// Binary Signal Boundary\n// Parse the provided input and return the answer as a string.\n#include <string>\nusing namespace std;\n\nclass Solution {\npublic:\n    string solve(const string& input) {\n        return \"\";\n    }\n};\n","go":"// Binary Signal Boundary\n// Parse stdin and print the answer.\npackage main\n\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"os\"\n)\n\nfunc Solve(input string) string {\n\treturn \"\"\n}\n\nfunc main() {\n\tinput, err := io.ReadAll(os.Stdin)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Print(Solve(string(input)))\n}\n","rust":"// Binary Signal Boundary\n// Parse the provided input and return the answer as a string.\npub fn solve(input: &str) -> String {\n    String::new()\n}\n","zig":"// Binary Signal Boundary\n// Parse stdin and print the answer.\nconst std = @import(\"std\");\n\nfn solve(input: []const u8, allocator: std.mem.Allocator) ![]u8 {\n    _ = allocator;\n    _ = input;\n    return \"\";\n}\n\npub fn main() !void {\n    var gpa = std.heap.GeneralPurposeAllocator(.{}){};\n    defer _ = gpa.deinit();\n    const allocator = gpa.allocator();\n\n    var stdin = std.io.getStdIn().reader();\n    const input = try stdin.readAllAlloc(allocator, 1 << 20);\n    defer allocator.free(input);\n\n    const output = try solve(input, allocator);\n    defer allocator.free(output);\n\n    try std.io.getStdOut().writer().print(\"{s}\", .{output});\n}\n","csharp":"// Binary Signal Boundary\n// Parse stdin and print the answer.\nusing System;\n\nclass Program\n{\n    static string Solve(string input)\n    {\n        return \"\";\n    }\n\n    static void Main()\n    {\n        var input = Console.In.ReadToEnd();\n        Console.Write(Solve(input));\n    }\n}\n","c":"// Binary Signal Boundary\n// Parse stdin and print the answer.\n#include <stdio.h>\n#include <stdlib.h>\n\nchar* solve(const char* input) {\n    (void)input;\n    char* out = malloc(1);\n    out[0] = ''\\0'';\n    return out;\n}\n\nint main(void) {\n    char* input = NULL;\n    size_t cap = 0;\n    size_t len = 0;\n\n    int ch;\n    while ((ch = getchar()) != EOF) {\n        if (len + 1 >= cap) {\n            cap = cap == 0 ? 256 : cap * 2;\n            input = realloc(input, cap);\n        }\n        input[len++] = (char)ch;\n    }\n    if (!input) {\n        input = malloc(1);\n    }\n    input[len] = ''\\0'';\n\n    char* out = solve(input);\n    fputs(out, stdout);\n\n    free(out);\n    free(input);\n    return 0;\n}\n","ruby":"# Binary Signal Boundary\n# Parse stdin and print the answer.\ndef solve(input)\n  \"\"\nend\n\ninput = STDIN.read\nprint solve(input)\n","php":"<?php\n// Binary Signal Boundary\n// Parse stdin and print the answer.\nfunction solve(string $input): string {\n    return \"\";\n}\n\n$input = stream_get_contents(STDIN);\necho solve($input);\n"},"acceptanceRate":70,"estimatedMinutes":18,"attempts":15100,"popularity":84,"createdAt":"2026-03-02","progress":"not-started","testCases":[{"name":"Case 1","input":"[1, 2, 2, 5, 9]\n5","expectedOutput":"3"}],"languageVersions":{"typescript":"5.8","javascript":"node22","python":"3.12","java":"21","cpp":"gnu++23","go":"1.24","rust":"1.80","zig":"0.13","csharp":"12.0","c":"c23","ruby":"3.3","php":"8.3"}}'::jsonb,
    '2026-03-02T00:00:00Z'::timestamptz,
    now()
)
on conflict (slug) do update set
    title = excluded.title,
    statement_md = excluded.statement_md,
    difficulty = excluded.difficulty,
    time_limit_ms = excluded.time_limit_ms,
    memory_limit_kb = excluded.memory_limit_kb,
    is_active = true,
    frontend_payload = excluded.frontend_payload,
    updated_at = now();
update task_languages
set is_enabled = false
where task_id = 'c88a7952-5f98-481a-a2f1-30fd7d49150b'::uuid
  and language not in ('typescript', 'javascript', 'python', 'java', 'cpp', 'go', 'rust', 'zig', 'csharp', 'c', 'ruby', 'php');
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '764b6b58-b740-439f-9299-97343f08de5b',
    'c88a7952-5f98-481a-a2f1-30fd7d49150b'::uuid,
    'typescript',
    '5.8',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '93d246b8-4849-4683-bc13-ad19240b35a6',
    'c88a7952-5f98-481a-a2f1-30fd7d49150b'::uuid,
    'javascript',
    'node22',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'e6c1a8c7-28f2-4026-8743-c1376bcf8078',
    'c88a7952-5f98-481a-a2f1-30fd7d49150b'::uuid,
    'python',
    '3.12',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '094224e2-edf4-43e1-9690-58939fac5dde',
    'c88a7952-5f98-481a-a2f1-30fd7d49150b'::uuid,
    'java',
    '21',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'b9c86e46-5033-4d86-8d19-b83ca3d22add',
    'c88a7952-5f98-481a-a2f1-30fd7d49150b'::uuid,
    'cpp',
    'gnu++23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'f1d9d5a9-f3df-4995-a6a1-1cc7ef3ca4bb',
    'c88a7952-5f98-481a-a2f1-30fd7d49150b'::uuid,
    'go',
    '1.24',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '057c202d-1e7d-49b7-afc6-a9fcd010551d',
    'c88a7952-5f98-481a-a2f1-30fd7d49150b'::uuid,
    'rust',
    '1.80',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '452086c6-61f9-47dc-bfee-510e5b4e915e',
    'c88a7952-5f98-481a-a2f1-30fd7d49150b'::uuid,
    'zig',
    '0.13',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'f1d654ca-3d9c-4c8d-b069-659aeb2a49c4',
    'c88a7952-5f98-481a-a2f1-30fd7d49150b'::uuid,
    'csharp',
    '12.0',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'a1e8d736-0600-4baa-be3a-0163502e9e29',
    'c88a7952-5f98-481a-a2f1-30fd7d49150b'::uuid,
    'c',
    'c23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '82f0e407-ea9b-48d0-ab7e-24d9ba15a47a',
    'c88a7952-5f98-481a-a2f1-30fd7d49150b'::uuid,
    'ruby',
    '3.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '797469c9-56bb-4e62-b240-731b3957d263',
    'c88a7952-5f98-481a-a2f1-30fd7d49150b'::uuid,
    'php',
    '8.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    'a273c101-3207-441a-826c-e4d64348d774',
    'c88a7952-5f98-481a-a2f1-30fd7d49150b'::uuid,
    'sample',
    1,
    '[1, 2, 2, 5, 9]
5',
    '3',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
update task_test_cases
set is_active = false
where task_id = 'c88a7952-5f98-481a-a2f1-30fd7d49150b'::uuid
  and kind = 'sample'
  and order_no not in (1);

-- archive-compression
insert into tasks (
    id, slug, title, statement_md, difficulty, time_limit_ms, memory_limit_kb, is_active, frontend_payload, created_at, updated_at
) values (
    '0ef6bc55-5aee-4c07-81de-5c1a96350c40',
    'archive-compression',
    'Archive Compression',
    'An archive stream contains lowercase block markers. Whenever k equal markers become adjacent, those k markers are compressed away.

Return the final stream after repeatedly applying compression until no more groups can be removed.',
    'medium',
    3000,
    262144,
    true,
    '{"id":"challenge-009","slug":"archive-compression","title":"Archive Compression","difficulty":"Medium","summary":"Remove adjacent duplicate blocks until the archive stream is stable.","description":["An archive stream contains lowercase block markers. Whenever k equal markers become adjacent, those k markers are compressed away.","Return the final stream after repeatedly applying compression until no more groups can be removed."],"examples":[{"title":"Chain reaction","input":"stream = \"abbbaaacc\", k = 3","output":"a","explanation":"bbb and aaa are removed, then cc remains with a single leading a."}],"constraints":["1 <= stream.length <= 200,000","2 <= k <= 10,000","stream contains lowercase English letters."],"notes":["A stack of character counts avoids rescanning the stream."],"tags":["Stack","String"],"category":"Strings","kind":"algorithmic","supportedLanguages":["typescript","javascript","python","java","cpp","go","rust","zig","csharp","c","ruby","php"],"starterCodeByLanguage":{"typescript":"// Archive Compression\n// Parse the provided input and return the answer as a string.\nexport function solve(input: string): string {\n  return \"\";\n}\n","javascript":"// Archive Compression\n// Parse the provided input and return the answer as a string.\nfunction solve(input) {\n  return \"\";\n}\n\nmodule.exports = { solve };\n","python":"# Archive Compression\n# Parse the provided input and return the answer as a string.\ndef solve(input_data: str) -> str:\n    return \"\"\n","java":"// Archive Compression\n// Parse the provided input and return the answer as a string.\nclass Solution {\n    public String solve(String input) {\n        return \"\";\n    }\n}\n","cpp":"// Archive Compression\n// Parse the provided input and return the answer as a string.\n#include <string>\nusing namespace std;\n\nclass Solution {\npublic:\n    string solve(const string& input) {\n        return \"\";\n    }\n};\n","go":"// Archive Compression\n// Parse stdin and print the answer.\npackage main\n\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"os\"\n)\n\nfunc Solve(input string) string {\n\treturn \"\"\n}\n\nfunc main() {\n\tinput, err := io.ReadAll(os.Stdin)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Print(Solve(string(input)))\n}\n","rust":"// Archive Compression\n// Parse the provided input and return the answer as a string.\npub fn solve(input: &str) -> String {\n    String::new()\n}\n","zig":"// Archive Compression\n// Parse stdin and print the answer.\nconst std = @import(\"std\");\n\nfn solve(input: []const u8, allocator: std.mem.Allocator) ![]u8 {\n    _ = allocator;\n    _ = input;\n    return \"\";\n}\n\npub fn main() !void {\n    var gpa = std.heap.GeneralPurposeAllocator(.{}){};\n    defer _ = gpa.deinit();\n    const allocator = gpa.allocator();\n\n    var stdin = std.io.getStdIn().reader();\n    const input = try stdin.readAllAlloc(allocator, 1 << 20);\n    defer allocator.free(input);\n\n    const output = try solve(input, allocator);\n    defer allocator.free(output);\n\n    try std.io.getStdOut().writer().print(\"{s}\", .{output});\n}\n","csharp":"// Archive Compression\n// Parse stdin and print the answer.\nusing System;\n\nclass Program\n{\n    static string Solve(string input)\n    {\n        return \"\";\n    }\n\n    static void Main()\n    {\n        var input = Console.In.ReadToEnd();\n        Console.Write(Solve(input));\n    }\n}\n","c":"// Archive Compression\n// Parse stdin and print the answer.\n#include <stdio.h>\n#include <stdlib.h>\n\nchar* solve(const char* input) {\n    (void)input;\n    char* out = malloc(1);\n    out[0] = ''\\0'';\n    return out;\n}\n\nint main(void) {\n    char* input = NULL;\n    size_t cap = 0;\n    size_t len = 0;\n\n    int ch;\n    while ((ch = getchar()) != EOF) {\n        if (len + 1 >= cap) {\n            cap = cap == 0 ? 256 : cap * 2;\n            input = realloc(input, cap);\n        }\n        input[len++] = (char)ch;\n    }\n    if (!input) {\n        input = malloc(1);\n    }\n    input[len] = ''\\0'';\n\n    char* out = solve(input);\n    fputs(out, stdout);\n\n    free(out);\n    free(input);\n    return 0;\n}\n","ruby":"# Archive Compression\n# Parse stdin and print the answer.\ndef solve(input)\n  \"\"\nend\n\ninput = STDIN.read\nprint solve(input)\n","php":"<?php\n// Archive Compression\n// Parse stdin and print the answer.\nfunction solve(string $input): string {\n    return \"\";\n}\n\n$input = stream_get_contents(STDIN);\necho solve($input);\n"},"acceptanceRate":52,"estimatedMinutes":28,"attempts":19480,"popularity":87,"createdAt":"2026-03-29","progress":"in-progress","testCases":[{"name":"Case 1","input":"abbbaaacc\n3","expectedOutput":"a"}],"languageVersions":{"typescript":"5.8","javascript":"node22","python":"3.12","java":"21","cpp":"gnu++23","go":"1.24","rust":"1.80","zig":"0.13","csharp":"12.0","c":"c23","ruby":"3.3","php":"8.3"}}'::jsonb,
    '2026-03-29T00:00:00Z'::timestamptz,
    now()
)
on conflict (slug) do update set
    title = excluded.title,
    statement_md = excluded.statement_md,
    difficulty = excluded.difficulty,
    time_limit_ms = excluded.time_limit_ms,
    memory_limit_kb = excluded.memory_limit_kb,
    is_active = true,
    frontend_payload = excluded.frontend_payload,
    updated_at = now();
update task_languages
set is_enabled = false
where task_id = '0ef6bc55-5aee-4c07-81de-5c1a96350c40'::uuid
  and language not in ('typescript', 'javascript', 'python', 'java', 'cpp', 'go', 'rust', 'zig', 'csharp', 'c', 'ruby', 'php');
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '4ed2c116-2d45-454f-9a62-c8b91d594541',
    '0ef6bc55-5aee-4c07-81de-5c1a96350c40'::uuid,
    'typescript',
    '5.8',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '90e4db5a-b9fa-48e8-ba0c-423bf10a2caa',
    '0ef6bc55-5aee-4c07-81de-5c1a96350c40'::uuid,
    'javascript',
    'node22',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '29edb8f1-c9e7-4f0b-b0b1-91a7c4e89549',
    '0ef6bc55-5aee-4c07-81de-5c1a96350c40'::uuid,
    'python',
    '3.12',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '326727a7-4a68-4616-a421-2c6827a494cd',
    '0ef6bc55-5aee-4c07-81de-5c1a96350c40'::uuid,
    'java',
    '21',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '17c79588-5dd9-4427-9f58-e7c196585f71',
    '0ef6bc55-5aee-4c07-81de-5c1a96350c40'::uuid,
    'cpp',
    'gnu++23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '0393523f-771e-48d4-b42a-ce70b750c2fd',
    '0ef6bc55-5aee-4c07-81de-5c1a96350c40'::uuid,
    'go',
    '1.24',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'f51242a9-290f-444d-b18c-94fe5ed342ac',
    '0ef6bc55-5aee-4c07-81de-5c1a96350c40'::uuid,
    'rust',
    '1.80',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'c6a8bacb-f144-4ce5-a7f2-87d4300699fe',
    '0ef6bc55-5aee-4c07-81de-5c1a96350c40'::uuid,
    'zig',
    '0.13',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'e1a4d01c-921c-44cd-80c5-ceb72a105c00',
    '0ef6bc55-5aee-4c07-81de-5c1a96350c40'::uuid,
    'csharp',
    '12.0',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '1da44521-9274-4544-abd8-a2b1e3f29e19',
    '0ef6bc55-5aee-4c07-81de-5c1a96350c40'::uuid,
    'c',
    'c23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '2772f6b3-df88-40db-91c7-c43967f54a5f',
    '0ef6bc55-5aee-4c07-81de-5c1a96350c40'::uuid,
    'ruby',
    '3.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'd4fa3016-b9c8-4049-ac07-f82298177263',
    '0ef6bc55-5aee-4c07-81de-5c1a96350c40'::uuid,
    'php',
    '8.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    '307e0fcc-f2fd-45ca-8722-f6823ebb91a5',
    '0ef6bc55-5aee-4c07-81de-5c1a96350c40'::uuid,
    'sample',
    1,
    'abbbaaacc
3',
    'a',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
update task_test_cases
set is_active = false
where task_id = '0ef6bc55-5aee-4c07-81de-5c1a96350c40'::uuid
  and kind = 'sample'
  and order_no not in (1);

-- colony-perimeter
insert into tasks (
    id, slug, title, statement_md, difficulty, time_limit_ms, memory_limit_kb, is_active, frontend_payload, created_at, updated_at
) values (
    '5030eae0-7a2f-44e5-af4f-f3afbc14dab5',
    'colony-perimeter',
    'Colony Perimeter',
    'A research grid marks occupied colony cells with 1 and empty cells with 0. Cells connect horizontally and vertically.

Return the total perimeter exposed to empty space or the grid boundary across all occupied cells.',
    'medium',
    3000,
    262144,
    true,
    '{"id":"challenge-010","slug":"colony-perimeter","title":"Colony Perimeter","difficulty":"Medium","summary":"Measure the exposed edge length of connected occupied cells.","description":["A research grid marks occupied colony cells with 1 and empty cells with 0. Cells connect horizontally and vertically.","Return the total perimeter exposed to empty space or the grid boundary across all occupied cells."],"examples":[{"title":"Small colony","input":"grid = [[1,1,0],[1,0,0],[1,1,1]]","output":"14"}],"constraints":["1 <= rows, columns <= 500","grid[r][c] is 0 or 1","The grid may contain multiple disconnected colonies."],"notes":["Each occupied cell starts with four edges; subtract shared edges."],"tags":["Graph","DFS","Simulation"],"category":"Graphs","kind":"algorithmic","supportedLanguages":["typescript","javascript","python","java","cpp","go","rust","zig","csharp","c","ruby","php"],"starterCodeByLanguage":{"typescript":"// Colony Perimeter\n// Parse the provided input and return the answer as a string.\nexport function solve(input: string): string {\n  return \"\";\n}\n","javascript":"// Colony Perimeter\n// Parse the provided input and return the answer as a string.\nfunction solve(input) {\n  return \"\";\n}\n\nmodule.exports = { solve };\n","python":"# Colony Perimeter\n# Parse the provided input and return the answer as a string.\ndef solve(input_data: str) -> str:\n    return \"\"\n","java":"// Colony Perimeter\n// Parse the provided input and return the answer as a string.\nclass Solution {\n    public String solve(String input) {\n        return \"\";\n    }\n}\n","cpp":"// Colony Perimeter\n// Parse the provided input and return the answer as a string.\n#include <string>\nusing namespace std;\n\nclass Solution {\npublic:\n    string solve(const string& input) {\n        return \"\";\n    }\n};\n","go":"// Colony Perimeter\n// Parse stdin and print the answer.\npackage main\n\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"os\"\n)\n\nfunc Solve(input string) string {\n\treturn \"\"\n}\n\nfunc main() {\n\tinput, err := io.ReadAll(os.Stdin)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Print(Solve(string(input)))\n}\n","rust":"// Colony Perimeter\n// Parse the provided input and return the answer as a string.\npub fn solve(input: &str) -> String {\n    String::new()\n}\n","zig":"// Colony Perimeter\n// Parse stdin and print the answer.\nconst std = @import(\"std\");\n\nfn solve(input: []const u8, allocator: std.mem.Allocator) ![]u8 {\n    _ = allocator;\n    _ = input;\n    return \"\";\n}\n\npub fn main() !void {\n    var gpa = std.heap.GeneralPurposeAllocator(.{}){};\n    defer _ = gpa.deinit();\n    const allocator = gpa.allocator();\n\n    var stdin = std.io.getStdIn().reader();\n    const input = try stdin.readAllAlloc(allocator, 1 << 20);\n    defer allocator.free(input);\n\n    const output = try solve(input, allocator);\n    defer allocator.free(output);\n\n    try std.io.getStdOut().writer().print(\"{s}\", .{output});\n}\n","csharp":"// Colony Perimeter\n// Parse stdin and print the answer.\nusing System;\n\nclass Program\n{\n    static string Solve(string input)\n    {\n        return \"\";\n    }\n\n    static void Main()\n    {\n        var input = Console.In.ReadToEnd();\n        Console.Write(Solve(input));\n    }\n}\n","c":"// Colony Perimeter\n// Parse stdin and print the answer.\n#include <stdio.h>\n#include <stdlib.h>\n\nchar* solve(const char* input) {\n    (void)input;\n    char* out = malloc(1);\n    out[0] = ''\\0'';\n    return out;\n}\n\nint main(void) {\n    char* input = NULL;\n    size_t cap = 0;\n    size_t len = 0;\n\n    int ch;\n    while ((ch = getchar()) != EOF) {\n        if (len + 1 >= cap) {\n            cap = cap == 0 ? 256 : cap * 2;\n            input = realloc(input, cap);\n        }\n        input[len++] = (char)ch;\n    }\n    if (!input) {\n        input = malloc(1);\n    }\n    input[len] = ''\\0'';\n\n    char* out = solve(input);\n    fputs(out, stdout);\n\n    free(out);\n    free(input);\n    return 0;\n}\n","ruby":"# Colony Perimeter\n# Parse stdin and print the answer.\ndef solve(input)\n  \"\"\nend\n\ninput = STDIN.read\nprint solve(input)\n","php":"<?php\n// Colony Perimeter\n// Parse stdin and print the answer.\nfunction solve(string $input): string {\n    return \"\";\n}\n\n$input = stream_get_contents(STDIN);\necho solve($input);\n"},"acceptanceRate":67,"estimatedMinutes":22,"attempts":11430,"popularity":78,"createdAt":"2026-02-18","progress":"not-started","testCases":[{"name":"Case 1","input":"[[1,1,0],[1,0,0],[1,1,1]]","expectedOutput":"14"}],"languageVersions":{"typescript":"5.8","javascript":"node22","python":"3.12","java":"21","cpp":"gnu++23","go":"1.24","rust":"1.80","zig":"0.13","csharp":"12.0","c":"c23","ruby":"3.3","php":"8.3"}}'::jsonb,
    '2026-02-18T00:00:00Z'::timestamptz,
    now()
)
on conflict (slug) do update set
    title = excluded.title,
    statement_md = excluded.statement_md,
    difficulty = excluded.difficulty,
    time_limit_ms = excluded.time_limit_ms,
    memory_limit_kb = excluded.memory_limit_kb,
    is_active = true,
    frontend_payload = excluded.frontend_payload,
    updated_at = now();
update task_languages
set is_enabled = false
where task_id = '5030eae0-7a2f-44e5-af4f-f3afbc14dab5'::uuid
  and language not in ('typescript', 'javascript', 'python', 'java', 'cpp', 'go', 'rust', 'zig', 'csharp', 'c', 'ruby', 'php');
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '85fe95d0-cc16-434f-8c19-f18aa179566d',
    '5030eae0-7a2f-44e5-af4f-f3afbc14dab5'::uuid,
    'typescript',
    '5.8',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'e7d3891c-4d36-41d7-96e8-cc78af81b3af',
    '5030eae0-7a2f-44e5-af4f-f3afbc14dab5'::uuid,
    'javascript',
    'node22',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '9dff626f-66e3-43f1-920a-7ec98bf151aa',
    '5030eae0-7a2f-44e5-af4f-f3afbc14dab5'::uuid,
    'python',
    '3.12',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '9c6ac9ac-9a3d-4186-a0e0-bf2e24fb9278',
    '5030eae0-7a2f-44e5-af4f-f3afbc14dab5'::uuid,
    'java',
    '21',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'dadd6890-05f1-4eb3-8308-88cbd4a5c31e',
    '5030eae0-7a2f-44e5-af4f-f3afbc14dab5'::uuid,
    'cpp',
    'gnu++23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '3bbc5e9c-c6d7-4b4f-8981-42c4238553a0',
    '5030eae0-7a2f-44e5-af4f-f3afbc14dab5'::uuid,
    'go',
    '1.24',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'de635d48-05a3-4de9-89bf-76ad6b52e9c5',
    '5030eae0-7a2f-44e5-af4f-f3afbc14dab5'::uuid,
    'rust',
    '1.80',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '90119bdb-9c70-4f9a-8f71-3627e433b27b',
    '5030eae0-7a2f-44e5-af4f-f3afbc14dab5'::uuid,
    'zig',
    '0.13',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '8e25080e-6c61-489d-89d4-b1ff90a02436',
    '5030eae0-7a2f-44e5-af4f-f3afbc14dab5'::uuid,
    'csharp',
    '12.0',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '156367d0-0fd9-47b2-a39c-73fef36ee4c5',
    '5030eae0-7a2f-44e5-af4f-f3afbc14dab5'::uuid,
    'c',
    'c23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'cd9589b6-3c1f-4315-bcb3-c6816dd5e94d',
    '5030eae0-7a2f-44e5-af4f-f3afbc14dab5'::uuid,
    'ruby',
    '3.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '2320e8b0-8975-4fa7-807a-5dfeab5908aa',
    '5030eae0-7a2f-44e5-af4f-f3afbc14dab5'::uuid,
    'php',
    '8.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    '491ec844-1281-423e-aea0-781c3ce896fe',
    '5030eae0-7a2f-44e5-af4f-f3afbc14dab5'::uuid,
    'sample',
    1,
    '[[1,1,0],[1,0,0],[1,1,1]]',
    '14',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
update task_test_cases
set is_active = false
where task_id = '5030eae0-7a2f-44e5-af4f-f3afbc14dab5'::uuid
  and kind = 'sample'
  and order_no not in (1);

-- latency-budget
insert into tasks (
    id, slug, title, statement_md, difficulty, time_limit_ms, memory_limit_kb, is_active, frontend_payload, created_at, updated_at
) values (
    '378ff817-0bf9-4dd6-aa0b-184855f9e023',
    'latency-budget',
    'Latency Budget',
    'A request can pass through optional services. Each service has a latency cost and a value score. You have a fixed latency budget.

Return the maximum total score achievable without exceeding the budget. Each service can be used at most once.',
    'hard',
    5000,
    393216,
    true,
    '{"id":"challenge-011","slug":"latency-budget","title":"Latency Budget","difficulty":"Hard","summary":"Maximize completed services without exceeding a total latency budget.","description":["A request can pass through optional services. Each service has a latency cost and a value score. You have a fixed latency budget.","Return the maximum total score achievable without exceeding the budget. Each service can be used at most once."],"examples":[{"title":"Budgeted selection","input":"budget = 7, services = [[3, 6], [4, 7], [5, 8]]","output":"13","explanation":"Use the services with costs 3 and 4 for a total score of 13."}],"constraints":["1 <= services.length <= 2,000","1 <= budget <= 20,000","1 <= latencyCost, valueScore <= 10,000"],"notes":["A one-dimensional dynamic programming table keeps memory manageable."],"tags":["Dynamic Programming","Array"],"category":"Optimization","kind":"algorithmic","supportedLanguages":["typescript","javascript","python","java","cpp","go","rust","zig","csharp","c","ruby","php"],"starterCodeByLanguage":{"typescript":"// Latency Budget\n// Parse the provided input and return the answer as a string.\nexport function solve(input: string): string {\n  return \"\";\n}\n","javascript":"// Latency Budget\n// Parse the provided input and return the answer as a string.\nfunction solve(input) {\n  return \"\";\n}\n\nmodule.exports = { solve };\n","python":"# Latency Budget\n# Parse the provided input and return the answer as a string.\ndef solve(input_data: str) -> str:\n    return \"\"\n","java":"// Latency Budget\n// Parse the provided input and return the answer as a string.\nclass Solution {\n    public String solve(String input) {\n        return \"\";\n    }\n}\n","cpp":"// Latency Budget\n// Parse the provided input and return the answer as a string.\n#include <string>\nusing namespace std;\n\nclass Solution {\npublic:\n    string solve(const string& input) {\n        return \"\";\n    }\n};\n","go":"// Latency Budget\n// Parse stdin and print the answer.\npackage main\n\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"os\"\n)\n\nfunc Solve(input string) string {\n\treturn \"\"\n}\n\nfunc main() {\n\tinput, err := io.ReadAll(os.Stdin)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Print(Solve(string(input)))\n}\n","rust":"// Latency Budget\n// Parse the provided input and return the answer as a string.\npub fn solve(input: &str) -> String {\n    String::new()\n}\n","zig":"// Latency Budget\n// Parse stdin and print the answer.\nconst std = @import(\"std\");\n\nfn solve(input: []const u8, allocator: std.mem.Allocator) ![]u8 {\n    _ = allocator;\n    _ = input;\n    return \"\";\n}\n\npub fn main() !void {\n    var gpa = std.heap.GeneralPurposeAllocator(.{}){};\n    defer _ = gpa.deinit();\n    const allocator = gpa.allocator();\n\n    var stdin = std.io.getStdIn().reader();\n    const input = try stdin.readAllAlloc(allocator, 1 << 20);\n    defer allocator.free(input);\n\n    const output = try solve(input, allocator);\n    defer allocator.free(output);\n\n    try std.io.getStdOut().writer().print(\"{s}\", .{output});\n}\n","csharp":"// Latency Budget\n// Parse stdin and print the answer.\nusing System;\n\nclass Program\n{\n    static string Solve(string input)\n    {\n        return \"\";\n    }\n\n    static void Main()\n    {\n        var input = Console.In.ReadToEnd();\n        Console.Write(Solve(input));\n    }\n}\n","c":"// Latency Budget\n// Parse stdin and print the answer.\n#include <stdio.h>\n#include <stdlib.h>\n\nchar* solve(const char* input) {\n    (void)input;\n    char* out = malloc(1);\n    out[0] = ''\\0'';\n    return out;\n}\n\nint main(void) {\n    char* input = NULL;\n    size_t cap = 0;\n    size_t len = 0;\n\n    int ch;\n    while ((ch = getchar()) != EOF) {\n        if (len + 1 >= cap) {\n            cap = cap == 0 ? 256 : cap * 2;\n            input = realloc(input, cap);\n        }\n        input[len++] = (char)ch;\n    }\n    if (!input) {\n        input = malloc(1);\n    }\n    input[len] = ''\\0'';\n\n    char* out = solve(input);\n    fputs(out, stdout);\n\n    free(out);\n    free(input);\n    return 0;\n}\n","ruby":"# Latency Budget\n# Parse stdin and print the answer.\ndef solve(input)\n  \"\"\nend\n\ninput = STDIN.read\nprint solve(input)\n","php":"<?php\n// Latency Budget\n// Parse stdin and print the answer.\nfunction solve(string $input): string {\n    return \"\";\n}\n\n$input = stream_get_contents(STDIN);\necho solve($input);\n"},"acceptanceRate":41,"estimatedMinutes":42,"attempts":9310,"popularity":81,"createdAt":"2026-03-31","progress":"not-started","testCases":[{"name":"Case 1","input":"7\n3 6\n4 7\n5 8","expectedOutput":"13"}],"languageVersions":{"typescript":"5.8","javascript":"node22","python":"3.12","java":"21","cpp":"gnu++23","go":"1.24","rust":"1.80","zig":"0.13","csharp":"12.0","c":"c23","ruby":"3.3","php":"8.3"}}'::jsonb,
    '2026-03-31T00:00:00Z'::timestamptz,
    now()
)
on conflict (slug) do update set
    title = excluded.title,
    statement_md = excluded.statement_md,
    difficulty = excluded.difficulty,
    time_limit_ms = excluded.time_limit_ms,
    memory_limit_kb = excluded.memory_limit_kb,
    is_active = true,
    frontend_payload = excluded.frontend_payload,
    updated_at = now();
update task_languages
set is_enabled = false
where task_id = '378ff817-0bf9-4dd6-aa0b-184855f9e023'::uuid
  and language not in ('typescript', 'javascript', 'python', 'java', 'cpp', 'go', 'rust', 'zig', 'csharp', 'c', 'ruby', 'php');
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '9b833cdf-0dd8-409f-a5b9-2743fa2e97af',
    '378ff817-0bf9-4dd6-aa0b-184855f9e023'::uuid,
    'typescript',
    '5.8',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '6263c674-af5c-461c-98dc-309b8b62861a',
    '378ff817-0bf9-4dd6-aa0b-184855f9e023'::uuid,
    'javascript',
    'node22',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '38b7dda1-95f9-4cdf-b16c-7c2abf47afc1',
    '378ff817-0bf9-4dd6-aa0b-184855f9e023'::uuid,
    'python',
    '3.12',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'c9c10540-623e-48be-8319-4ad521a54d93',
    '378ff817-0bf9-4dd6-aa0b-184855f9e023'::uuid,
    'java',
    '21',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '36e772ca-cbd0-4b6d-9d47-30c7d7c4bb0f',
    '378ff817-0bf9-4dd6-aa0b-184855f9e023'::uuid,
    'cpp',
    'gnu++23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '39cdfa23-ccda-4907-8ea3-5982c7ce1d60',
    '378ff817-0bf9-4dd6-aa0b-184855f9e023'::uuid,
    'go',
    '1.24',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'b278d656-4ba4-4595-a4ee-cb71943d2daf',
    '378ff817-0bf9-4dd6-aa0b-184855f9e023'::uuid,
    'rust',
    '1.80',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '89aed21f-d9f8-42f8-812e-ca5b63381b83',
    '378ff817-0bf9-4dd6-aa0b-184855f9e023'::uuid,
    'zig',
    '0.13',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '0a0d96bf-14d9-4ce8-89a4-13b1b4a4bd1b',
    '378ff817-0bf9-4dd6-aa0b-184855f9e023'::uuid,
    'csharp',
    '12.0',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '3a69e20e-a539-4e8c-a0d8-945697cb63ca',
    '378ff817-0bf9-4dd6-aa0b-184855f9e023'::uuid,
    'c',
    'c23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'c6e23d96-330f-49ff-8001-9ab5f1c1f2e4',
    '378ff817-0bf9-4dd6-aa0b-184855f9e023'::uuid,
    'ruby',
    '3.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '1f1770ec-adae-4cb4-98c1-167b209162a4',
    '378ff817-0bf9-4dd6-aa0b-184855f9e023'::uuid,
    'php',
    '8.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    '9d6f0707-2c01-4070-9921-bc9548d5e169',
    '378ff817-0bf9-4dd6-aa0b-184855f9e023'::uuid,
    'sample',
    1,
    '7
3 6
4 7
5 8',
    '13',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
update task_test_cases
set is_active = false
where task_id = '378ff817-0bf9-4dd6-aa0b-184855f9e023'::uuid
  and kind = 'sample'
  and order_no not in (1);

-- quantum-safe-lock
insert into tasks (
    id, slug, title, statement_md, difficulty, time_limit_ms, memory_limit_kb, is_active, frontend_payload, created_at, updated_at
) values (
    '7cec64ee-0e0d-4211-8e51-2333428ecbf7',
    'quantum-safe-lock',
    'Quantum Safe Lock',
    'A lock validates access by scanning the latest n bits entered. You need a sequence that causes every possible n-bit pattern to appear at least once.

Return any shortest valid sequence over 0 and 1.',
    'hard',
    5000,
    393216,
    true,
    '{"id":"challenge-012","slug":"quantum-safe-lock","title":"Quantum Safe Lock","difficulty":"Hard","summary":"Generate the shortest unlock sequence covering every binary pattern.","description":["A lock validates access by scanning the latest n bits entered. You need a sequence that causes every possible n-bit pattern to appear at least once.","Return any shortest valid sequence over 0 and 1."],"examples":[{"title":"Two-bit lock","input":"n = 2","output":"00110","explanation":"The sequence contains 00, 01, 11, and 10."}],"constraints":["1 <= n <= 16","The answer length must be 2^n + n - 1.","Any shortest sequence is accepted."],"notes":["Think of states as bit patterns and edges as appended bits."],"tags":["Backtracking","Bit Manipulation","Graph"],"category":"Advanced","kind":"algorithmic","supportedLanguages":["typescript","javascript","python","java","cpp","go","rust","zig","csharp","c","ruby","php"],"starterCodeByLanguage":{"typescript":"// Quantum Safe Lock\n// Parse the provided input and return the answer as a string.\nexport function solve(input: string): string {\n  return \"\";\n}\n","javascript":"// Quantum Safe Lock\n// Parse the provided input and return the answer as a string.\nfunction solve(input) {\n  return \"\";\n}\n\nmodule.exports = { solve };\n","python":"# Quantum Safe Lock\n# Parse the provided input and return the answer as a string.\ndef solve(input_data: str) -> str:\n    return \"\"\n","java":"// Quantum Safe Lock\n// Parse the provided input and return the answer as a string.\nclass Solution {\n    public String solve(String input) {\n        return \"\";\n    }\n}\n","cpp":"// Quantum Safe Lock\n// Parse the provided input and return the answer as a string.\n#include <string>\nusing namespace std;\n\nclass Solution {\npublic:\n    string solve(const string& input) {\n        return \"\";\n    }\n};\n","go":"// Quantum Safe Lock\n// Parse stdin and print the answer.\npackage main\n\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"os\"\n)\n\nfunc Solve(input string) string {\n\treturn \"\"\n}\n\nfunc main() {\n\tinput, err := io.ReadAll(os.Stdin)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Print(Solve(string(input)))\n}\n","rust":"// Quantum Safe Lock\n// Parse the provided input and return the answer as a string.\npub fn solve(input: &str) -> String {\n    String::new()\n}\n","zig":"// Quantum Safe Lock\n// Parse stdin and print the answer.\nconst std = @import(\"std\");\n\nfn solve(input: []const u8, allocator: std.mem.Allocator) ![]u8 {\n    _ = allocator;\n    _ = input;\n    return \"\";\n}\n\npub fn main() !void {\n    var gpa = std.heap.GeneralPurposeAllocator(.{}){};\n    defer _ = gpa.deinit();\n    const allocator = gpa.allocator();\n\n    var stdin = std.io.getStdIn().reader();\n    const input = try stdin.readAllAlloc(allocator, 1 << 20);\n    defer allocator.free(input);\n\n    const output = try solve(input, allocator);\n    defer allocator.free(output);\n\n    try std.io.getStdOut().writer().print(\"{s}\", .{output});\n}\n","csharp":"// Quantum Safe Lock\n// Parse stdin and print the answer.\nusing System;\n\nclass Program\n{\n    static string Solve(string input)\n    {\n        return \"\";\n    }\n\n    static void Main()\n    {\n        var input = Console.In.ReadToEnd();\n        Console.Write(Solve(input));\n    }\n}\n","c":"// Quantum Safe Lock\n// Parse stdin and print the answer.\n#include <stdio.h>\n#include <stdlib.h>\n\nchar* solve(const char* input) {\n    (void)input;\n    char* out = malloc(1);\n    out[0] = ''\\0'';\n    return out;\n}\n\nint main(void) {\n    char* input = NULL;\n    size_t cap = 0;\n    size_t len = 0;\n\n    int ch;\n    while ((ch = getchar()) != EOF) {\n        if (len + 1 >= cap) {\n            cap = cap == 0 ? 256 : cap * 2;\n            input = realloc(input, cap);\n        }\n        input[len++] = (char)ch;\n    }\n    if (!input) {\n        input = malloc(1);\n    }\n    input[len] = ''\\0'';\n\n    char* out = solve(input);\n    fputs(out, stdout);\n\n    free(out);\n    free(input);\n    return 0;\n}\n","ruby":"# Quantum Safe Lock\n# Parse stdin and print the answer.\ndef solve(input)\n  \"\"\nend\n\ninput = STDIN.read\nprint solve(input)\n","php":"<?php\n// Quantum Safe Lock\n// Parse stdin and print the answer.\nfunction solve(string $input): string {\n    return \"\";\n}\n\n$input = stream_get_contents(STDIN);\necho solve($input);\n"},"acceptanceRate":33,"estimatedMinutes":50,"attempts":6020,"popularity":73,"createdAt":"2026-04-04","progress":"locked","testCases":[{"name":"Case 1","input":"2","expectedOutput":"00110"}],"languageVersions":{"typescript":"5.8","javascript":"node22","python":"3.12","java":"21","cpp":"gnu++23","go":"1.24","rust":"1.80","zig":"0.13","csharp":"12.0","c":"c23","ruby":"3.3","php":"8.3"}}'::jsonb,
    '2026-04-04T00:00:00Z'::timestamptz,
    now()
)
on conflict (slug) do update set
    title = excluded.title,
    statement_md = excluded.statement_md,
    difficulty = excluded.difficulty,
    time_limit_ms = excluded.time_limit_ms,
    memory_limit_kb = excluded.memory_limit_kb,
    is_active = true,
    frontend_payload = excluded.frontend_payload,
    updated_at = now();
update task_languages
set is_enabled = false
where task_id = '7cec64ee-0e0d-4211-8e51-2333428ecbf7'::uuid
  and language not in ('typescript', 'javascript', 'python', 'java', 'cpp', 'go', 'rust', 'zig', 'csharp', 'c', 'ruby', 'php');
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'bc5f066f-e0a9-4660-9a69-037adb190675',
    '7cec64ee-0e0d-4211-8e51-2333428ecbf7'::uuid,
    'typescript',
    '5.8',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '43548e08-49c5-4037-955b-c9a993708ac6',
    '7cec64ee-0e0d-4211-8e51-2333428ecbf7'::uuid,
    'javascript',
    'node22',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'bd60589a-0f3d-498e-b8be-556bf9649a0c',
    '7cec64ee-0e0d-4211-8e51-2333428ecbf7'::uuid,
    'python',
    '3.12',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'dc3782d4-4392-42d4-a5e3-60d66e7327b9',
    '7cec64ee-0e0d-4211-8e51-2333428ecbf7'::uuid,
    'java',
    '21',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'f183371d-5685-45b9-afe3-cf86ab84008e',
    '7cec64ee-0e0d-4211-8e51-2333428ecbf7'::uuid,
    'cpp',
    'gnu++23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '9cee04f7-0a9f-462c-9ce9-893d9060ba99',
    '7cec64ee-0e0d-4211-8e51-2333428ecbf7'::uuid,
    'go',
    '1.24',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '8e65d05f-9e3e-42bf-a4ed-4e255d2953df',
    '7cec64ee-0e0d-4211-8e51-2333428ecbf7'::uuid,
    'rust',
    '1.80',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '468cad70-619d-4e1c-b865-1d307ed5d144',
    '7cec64ee-0e0d-4211-8e51-2333428ecbf7'::uuid,
    'zig',
    '0.13',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '5677bdcc-5689-4435-970a-6e7c95c05c33',
    '7cec64ee-0e0d-4211-8e51-2333428ecbf7'::uuid,
    'csharp',
    '12.0',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'e252e465-53b6-4e6a-967c-2525f6db76db',
    '7cec64ee-0e0d-4211-8e51-2333428ecbf7'::uuid,
    'c',
    'c23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '21a15ca6-f791-4b31-b1ba-505d29c318e9',
    '7cec64ee-0e0d-4211-8e51-2333428ecbf7'::uuid,
    'ruby',
    '3.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '600ef35e-61a4-4a9a-bdf7-a6e8b4437aa2',
    '7cec64ee-0e0d-4211-8e51-2333428ecbf7'::uuid,
    'php',
    '8.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    '3506fdaf-b60d-478a-9be7-22ad2cd879ac',
    '7cec64ee-0e0d-4211-8e51-2333428ecbf7'::uuid,
    'sample',
    1,
    '2',
    '00110',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
update task_test_cases
set is_active = false
where task_id = '7cec64ee-0e0d-4211-8e51-2333428ecbf7'::uuid
  and kind = 'sample'
  and order_no not in (1);

-- priority-inbox
insert into tasks (
    id, slug, title, statement_md, difficulty, time_limit_ms, memory_limit_kb, is_active, frontend_payload, created_at, updated_at
) values (
    '87631b99-f73c-47bd-96f6-5bc7eeadc0d9',
    'priority-inbox',
    'Priority Inbox',
    'A support inbox stores tickets with a numeric priority. Higher numbers should be handled first.

Return the ids of the k highest priority tickets, ordered by priority descending and then by smaller id for ties.',
    'easy',
    2000,
    262144,
    true,
    '{"id":"challenge-013","slug":"priority-inbox","title":"Priority Inbox","difficulty":"Easy","summary":"Return the k highest priority tickets while preserving clear ordering.","description":["A support inbox stores tickets with a numeric priority. Higher numbers should be handled first.","Return the ids of the k highest priority tickets, ordered by priority descending and then by smaller id for ties."],"examples":[{"title":"Tie by id","input":"tickets = [[7, 40], [3, 50], [9, 50]], k = 2","output":"[3, 9]"}],"constraints":["1 <= tickets.length <= 100,000","1 <= k <= tickets.length","Ticket ids are unique positive integers."],"notes":["Sorting is acceptable; a heap is useful when k is much smaller than n."],"tags":["Heap","Sorting"],"category":"Data Structures","kind":"algorithmic","supportedLanguages":["typescript","javascript","python","java","cpp","go","rust","zig","csharp","c","ruby","php"],"starterCodeByLanguage":{"typescript":"// Priority Inbox\n// Parse the provided input and return the answer as a string.\nexport function solve(input: string): string {\n  return \"\";\n}\n","javascript":"// Priority Inbox\n// Parse the provided input and return the answer as a string.\nfunction solve(input) {\n  return \"\";\n}\n\nmodule.exports = { solve };\n","python":"# Priority Inbox\n# Parse the provided input and return the answer as a string.\ndef solve(input_data: str) -> str:\n    return \"\"\n","java":"// Priority Inbox\n// Parse the provided input and return the answer as a string.\nclass Solution {\n    public String solve(String input) {\n        return \"\";\n    }\n}\n","cpp":"// Priority Inbox\n// Parse the provided input and return the answer as a string.\n#include <string>\nusing namespace std;\n\nclass Solution {\npublic:\n    string solve(const string& input) {\n        return \"\";\n    }\n};\n","go":"// Priority Inbox\n// Parse stdin and print the answer.\npackage main\n\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"os\"\n)\n\nfunc Solve(input string) string {\n\treturn \"\"\n}\n\nfunc main() {\n\tinput, err := io.ReadAll(os.Stdin)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Print(Solve(string(input)))\n}\n","rust":"// Priority Inbox\n// Parse the provided input and return the answer as a string.\npub fn solve(input: &str) -> String {\n    String::new()\n}\n","zig":"// Priority Inbox\n// Parse stdin and print the answer.\nconst std = @import(\"std\");\n\nfn solve(input: []const u8, allocator: std.mem.Allocator) ![]u8 {\n    _ = allocator;\n    _ = input;\n    return \"\";\n}\n\npub fn main() !void {\n    var gpa = std.heap.GeneralPurposeAllocator(.{}){};\n    defer _ = gpa.deinit();\n    const allocator = gpa.allocator();\n\n    var stdin = std.io.getStdIn().reader();\n    const input = try stdin.readAllAlloc(allocator, 1 << 20);\n    defer allocator.free(input);\n\n    const output = try solve(input, allocator);\n    defer allocator.free(output);\n\n    try std.io.getStdOut().writer().print(\"{s}\", .{output});\n}\n","csharp":"// Priority Inbox\n// Parse stdin and print the answer.\nusing System;\n\nclass Program\n{\n    static string Solve(string input)\n    {\n        return \"\";\n    }\n\n    static void Main()\n    {\n        var input = Console.In.ReadToEnd();\n        Console.Write(Solve(input));\n    }\n}\n","c":"// Priority Inbox\n// Parse stdin and print the answer.\n#include <stdio.h>\n#include <stdlib.h>\n\nchar* solve(const char* input) {\n    (void)input;\n    char* out = malloc(1);\n    out[0] = ''\\0'';\n    return out;\n}\n\nint main(void) {\n    char* input = NULL;\n    size_t cap = 0;\n    size_t len = 0;\n\n    int ch;\n    while ((ch = getchar()) != EOF) {\n        if (len + 1 >= cap) {\n            cap = cap == 0 ? 256 : cap * 2;\n            input = realloc(input, cap);\n        }\n        input[len++] = (char)ch;\n    }\n    if (!input) {\n        input = malloc(1);\n    }\n    input[len] = ''\\0'';\n\n    char* out = solve(input);\n    fputs(out, stdout);\n\n    free(out);\n    free(input);\n    return 0;\n}\n","ruby":"# Priority Inbox\n# Parse stdin and print the answer.\ndef solve(input)\n  \"\"\nend\n\ninput = STDIN.read\nprint solve(input)\n","php":"<?php\n// Priority Inbox\n// Parse stdin and print the answer.\nfunction solve(string $input): string {\n    return \"\";\n}\n\n$input = stream_get_contents(STDIN);\necho solve($input);\n"},"acceptanceRate":74,"estimatedMinutes":16,"attempts":8360,"popularity":76,"createdAt":"2026-02-11","progress":"not-started","testCases":[{"name":"Case 1","input":"7 40\n3 50\n9 50\n2","expectedOutput":"[3, 9]"}],"languageVersions":{"typescript":"5.8","javascript":"node22","python":"3.12","java":"21","cpp":"gnu++23","go":"1.24","rust":"1.80","zig":"0.13","csharp":"12.0","c":"c23","ruby":"3.3","php":"8.3"}}'::jsonb,
    '2026-02-11T00:00:00Z'::timestamptz,
    now()
)
on conflict (slug) do update set
    title = excluded.title,
    statement_md = excluded.statement_md,
    difficulty = excluded.difficulty,
    time_limit_ms = excluded.time_limit_ms,
    memory_limit_kb = excluded.memory_limit_kb,
    is_active = true,
    frontend_payload = excluded.frontend_payload,
    updated_at = now();
update task_languages
set is_enabled = false
where task_id = '87631b99-f73c-47bd-96f6-5bc7eeadc0d9'::uuid
  and language not in ('typescript', 'javascript', 'python', 'java', 'cpp', 'go', 'rust', 'zig', 'csharp', 'c', 'ruby', 'php');
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'a69c87b7-3160-43a2-aa30-a472c8d19097',
    '87631b99-f73c-47bd-96f6-5bc7eeadc0d9'::uuid,
    'typescript',
    '5.8',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '8871063c-44c2-47cf-b448-fc9b26d6e682',
    '87631b99-f73c-47bd-96f6-5bc7eeadc0d9'::uuid,
    'javascript',
    'node22',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'a9ab48ff-22d8-4d56-8e54-986c56b6416b',
    '87631b99-f73c-47bd-96f6-5bc7eeadc0d9'::uuid,
    'python',
    '3.12',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '4b27c79f-65d4-4f72-90e6-51106f07f1be',
    '87631b99-f73c-47bd-96f6-5bc7eeadc0d9'::uuid,
    'java',
    '21',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'ecbd0ad2-846e-421b-9fa1-0d584225bdc6',
    '87631b99-f73c-47bd-96f6-5bc7eeadc0d9'::uuid,
    'cpp',
    'gnu++23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '9f4d363d-6e05-40e0-872a-8fbe0d43c568',
    '87631b99-f73c-47bd-96f6-5bc7eeadc0d9'::uuid,
    'go',
    '1.24',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'a5618dab-2a75-46be-81d3-da1f6a457975',
    '87631b99-f73c-47bd-96f6-5bc7eeadc0d9'::uuid,
    'rust',
    '1.80',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '2d077d90-0578-4aaf-84aa-312ff219b735',
    '87631b99-f73c-47bd-96f6-5bc7eeadc0d9'::uuid,
    'zig',
    '0.13',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '1ee42f6d-aed4-496b-b8ec-107ab0ba2351',
    '87631b99-f73c-47bd-96f6-5bc7eeadc0d9'::uuid,
    'csharp',
    '12.0',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '6f691891-8749-4525-88e9-bd4a1b7b6376',
    '87631b99-f73c-47bd-96f6-5bc7eeadc0d9'::uuid,
    'c',
    'c23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '0d9c2ec5-b833-4120-ae2d-49e4051c6ea9',
    '87631b99-f73c-47bd-96f6-5bc7eeadc0d9'::uuid,
    'ruby',
    '3.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '834cf1e6-a114-4507-bf52-165a45722235',
    '87631b99-f73c-47bd-96f6-5bc7eeadc0d9'::uuid,
    'php',
    '8.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    'c98f585e-6dda-4e38-999f-1b3ec32d8d12',
    '87631b99-f73c-47bd-96f6-5bc7eeadc0d9'::uuid,
    'sample',
    1,
    '7 40
3 50
9 50
2',
    '[3, 9]',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
update task_test_cases
set is_active = false
where task_id = '87631b99-f73c-47bd-96f6-5bc7eeadc0d9'::uuid
  and kind = 'sample'
  and order_no not in (1);

-- trade-ledger-balance
insert into tasks (
    id, slug, title, statement_md, difficulty, time_limit_ms, memory_limit_kb, is_active, frontend_payload, created_at, updated_at
) values (
    'f2dd347d-82ab-4b6f-a8cb-5e6ee41f5a2b',
    'trade-ledger-balance',
    'Trade Ledger Balance',
    'A ledger marks each transaction as +1 for an inflow and -1 for an outflow. A balanced period has a net sum of zero.

Return the length of the longest contiguous balanced period.',
    'medium',
    3000,
    262144,
    true,
    '{"id":"challenge-014","slug":"trade-ledger-balance","title":"Trade Ledger Balance","difficulty":"Medium","summary":"Find the longest period where account inflows and outflows balance.","description":["A ledger marks each transaction as +1 for an inflow and -1 for an outflow. A balanced period has a net sum of zero.","Return the length of the longest contiguous balanced period."],"examples":[{"title":"Long balanced span","input":"ledger = [1, -1, -1, 1, 1, -1]","output":"6"},{"title":"Partial span","input":"ledger = [1, 1, -1, 1]","output":"2"}],"constraints":["1 <= ledger.length <= 200,000","ledger[i] is either 1 or -1","Expected complexity is O(n)."],"notes":["Store the earliest index where each prefix sum appears."],"tags":["Prefix Sum","Hash Table","Array"],"category":"Data Structures","kind":"algorithmic","supportedLanguages":["typescript","javascript","python","java","cpp","go","rust","zig","csharp","c","ruby","php"],"starterCodeByLanguage":{"typescript":"// Trade Ledger Balance\n// Parse the provided input and return the answer as a string.\nexport function solve(input: string): string {\n  return \"\";\n}\n","javascript":"// Trade Ledger Balance\n// Parse the provided input and return the answer as a string.\nfunction solve(input) {\n  return \"\";\n}\n\nmodule.exports = { solve };\n","python":"# Trade Ledger Balance\n# Parse the provided input and return the answer as a string.\ndef solve(input_data: str) -> str:\n    return \"\"\n","java":"// Trade Ledger Balance\n// Parse the provided input and return the answer as a string.\nclass Solution {\n    public String solve(String input) {\n        return \"\";\n    }\n}\n","cpp":"// Trade Ledger Balance\n// Parse the provided input and return the answer as a string.\n#include <string>\nusing namespace std;\n\nclass Solution {\npublic:\n    string solve(const string& input) {\n        return \"\";\n    }\n};\n","go":"// Trade Ledger Balance\n// Parse stdin and print the answer.\npackage main\n\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"os\"\n)\n\nfunc Solve(input string) string {\n\treturn \"\"\n}\n\nfunc main() {\n\tinput, err := io.ReadAll(os.Stdin)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Print(Solve(string(input)))\n}\n","rust":"// Trade Ledger Balance\n// Parse the provided input and return the answer as a string.\npub fn solve(input: &str) -> String {\n    String::new()\n}\n","zig":"// Trade Ledger Balance\n// Parse stdin and print the answer.\nconst std = @import(\"std\");\n\nfn solve(input: []const u8, allocator: std.mem.Allocator) ![]u8 {\n    _ = allocator;\n    _ = input;\n    return \"\";\n}\n\npub fn main() !void {\n    var gpa = std.heap.GeneralPurposeAllocator(.{}){};\n    defer _ = gpa.deinit();\n    const allocator = gpa.allocator();\n\n    var stdin = std.io.getStdIn().reader();\n    const input = try stdin.readAllAlloc(allocator, 1 << 20);\n    defer allocator.free(input);\n\n    const output = try solve(input, allocator);\n    defer allocator.free(output);\n\n    try std.io.getStdOut().writer().print(\"{s}\", .{output});\n}\n","csharp":"// Trade Ledger Balance\n// Parse stdin and print the answer.\nusing System;\n\nclass Program\n{\n    static string Solve(string input)\n    {\n        return \"\";\n    }\n\n    static void Main()\n    {\n        var input = Console.In.ReadToEnd();\n        Console.Write(Solve(input));\n    }\n}\n","c":"// Trade Ledger Balance\n// Parse stdin and print the answer.\n#include <stdio.h>\n#include <stdlib.h>\n\nchar* solve(const char* input) {\n    (void)input;\n    char* out = malloc(1);\n    out[0] = ''\\0'';\n    return out;\n}\n\nint main(void) {\n    char* input = NULL;\n    size_t cap = 0;\n    size_t len = 0;\n\n    int ch;\n    while ((ch = getchar()) != EOF) {\n        if (len + 1 >= cap) {\n            cap = cap == 0 ? 256 : cap * 2;\n            input = realloc(input, cap);\n        }\n        input[len++] = (char)ch;\n    }\n    if (!input) {\n        input = malloc(1);\n    }\n    input[len] = ''\\0'';\n\n    char* out = solve(input);\n    fputs(out, stdout);\n\n    free(out);\n    free(input);\n    return 0;\n}\n","ruby":"# Trade Ledger Balance\n# Parse stdin and print the answer.\ndef solve(input)\n  \"\"\nend\n\ninput = STDIN.read\nprint solve(input)\n","php":"<?php\n// Trade Ledger Balance\n// Parse stdin and print the answer.\nfunction solve(string $input): string {\n    return \"\";\n}\n\n$input = stream_get_contents(STDIN);\necho solve($input);\n"},"acceptanceRate":59,"estimatedMinutes":25,"attempts":12380,"popularity":80,"createdAt":"2026-03-12","progress":"not-started","testCases":[{"name":"Case 1","input":"[1, -1, -1, 1, 1, -1]","expectedOutput":"6"}],"languageVersions":{"typescript":"5.8","javascript":"node22","python":"3.12","java":"21","cpp":"gnu++23","go":"1.24","rust":"1.80","zig":"0.13","csharp":"12.0","c":"c23","ruby":"3.3","php":"8.3"}}'::jsonb,
    '2026-03-12T00:00:00Z'::timestamptz,
    now()
)
on conflict (slug) do update set
    title = excluded.title,
    statement_md = excluded.statement_md,
    difficulty = excluded.difficulty,
    time_limit_ms = excluded.time_limit_ms,
    memory_limit_kb = excluded.memory_limit_kb,
    is_active = true,
    frontend_payload = excluded.frontend_payload,
    updated_at = now();
update task_languages
set is_enabled = false
where task_id = 'f2dd347d-82ab-4b6f-a8cb-5e6ee41f5a2b'::uuid
  and language not in ('typescript', 'javascript', 'python', 'java', 'cpp', 'go', 'rust', 'zig', 'csharp', 'c', 'ruby', 'php');
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '9e88c5e1-6534-4094-a009-8c74f7c42184',
    'f2dd347d-82ab-4b6f-a8cb-5e6ee41f5a2b'::uuid,
    'typescript',
    '5.8',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '51cbd439-fcfe-4c48-ac20-3d276ae77431',
    'f2dd347d-82ab-4b6f-a8cb-5e6ee41f5a2b'::uuid,
    'javascript',
    'node22',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '5e757aa1-dfe1-4883-bf2c-6c4214c2c49d',
    'f2dd347d-82ab-4b6f-a8cb-5e6ee41f5a2b'::uuid,
    'python',
    '3.12',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'dd5f7ea1-97a1-4358-a234-f59e2b869f5c',
    'f2dd347d-82ab-4b6f-a8cb-5e6ee41f5a2b'::uuid,
    'java',
    '21',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '87606e8e-3e43-4227-a3d5-a824406d0e86',
    'f2dd347d-82ab-4b6f-a8cb-5e6ee41f5a2b'::uuid,
    'cpp',
    'gnu++23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'acdaad6f-5254-41d7-9244-6cb5e0fc847f',
    'f2dd347d-82ab-4b6f-a8cb-5e6ee41f5a2b'::uuid,
    'go',
    '1.24',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '16f5a515-082a-447e-ba17-d942da6facf4',
    'f2dd347d-82ab-4b6f-a8cb-5e6ee41f5a2b'::uuid,
    'rust',
    '1.80',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '8ebdae6a-1bfe-471c-8e03-45fe5440db2a',
    'f2dd347d-82ab-4b6f-a8cb-5e6ee41f5a2b'::uuid,
    'zig',
    '0.13',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'fffa7116-042b-4297-90f1-896a513a9f15',
    'f2dd347d-82ab-4b6f-a8cb-5e6ee41f5a2b'::uuid,
    'csharp',
    '12.0',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'c554b636-ca83-4218-9ca0-e8a4d1439d00',
    'f2dd347d-82ab-4b6f-a8cb-5e6ee41f5a2b'::uuid,
    'c',
    'c23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'da6652f0-2c59-4640-85e0-e19cf1a1ba7a',
    'f2dd347d-82ab-4b6f-a8cb-5e6ee41f5a2b'::uuid,
    'ruby',
    '3.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '81f32c14-91f7-4113-9fa8-d04b7907fc49',
    'f2dd347d-82ab-4b6f-a8cb-5e6ee41f5a2b'::uuid,
    'php',
    '8.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    '7299dc8a-efa9-4a93-9141-28d20b259542',
    'f2dd347d-82ab-4b6f-a8cb-5e6ee41f5a2b'::uuid,
    'sample',
    1,
    '[1, -1, -1, 1, 1, -1]',
    '6',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
update task_test_cases
set is_active = false
where task_id = 'f2dd347d-82ab-4b6f-a8cb-5e6ee41f5a2b'::uuid
  and kind = 'sample'
  and order_no not in (1);

-- unioned-accounts
insert into tasks (
    id, slug, title, statement_md, difficulty, time_limit_ms, memory_limit_kb, is_active, frontend_payload, created_at, updated_at
) values (
    'd3755433-bd98-47eb-a734-904eee2713b2',
    'unioned-accounts',
    'Unioned Accounts',
    'Each account record contains a display name followed by one or more verified email addresses. Records with any shared email belong to the same person.

Return merged accounts with the display name from the first record in each group and sorted unique emails.',
    'medium',
    3000,
    262144,
    true,
    '{"id":"challenge-015","slug":"unioned-accounts","title":"Unioned Accounts","difficulty":"Medium","summary":"Merge account records that share at least one verified email address.","description":["Each account record contains a display name followed by one or more verified email addresses. Records with any shared email belong to the same person.","Return merged accounts with the display name from the first record in each group and sorted unique emails."],"examples":[{"title":"Shared email merge","input":"accounts = [[\"Rin\",\"r@cf.dev\",\"rin@cf.dev\"],[\"Kai\",\"k@cf.dev\"],[\"Rin\",\"rin@cf.dev\",\"ops@cf.dev\"]]","output":"[[\"Rin\",\"ops@cf.dev\",\"r@cf.dev\",\"rin@cf.dev\"],[\"Kai\",\"k@cf.dev\"]]"}],"constraints":["1 <= accounts.length <= 20,000","1 <= emails per account <= 20","Email strings are lowercase and unique within a single record."],"notes":["Union-Find keeps connected records cheap to merge."],"tags":["Union-Find","Hash Table","Sorting"],"category":"Data Structures","kind":"algorithmic","supportedLanguages":["typescript","javascript","python","java","cpp","go","rust","zig","csharp","c","ruby","php"],"starterCodeByLanguage":{"typescript":"// Unioned Accounts\n// Parse the provided input and return the answer as a string.\nexport function solve(input: string): string {\n  return \"\";\n}\n","javascript":"// Unioned Accounts\n// Parse the provided input and return the answer as a string.\nfunction solve(input) {\n  return \"\";\n}\n\nmodule.exports = { solve };\n","python":"# Unioned Accounts\n# Parse the provided input and return the answer as a string.\ndef solve(input_data: str) -> str:\n    return \"\"\n","java":"// Unioned Accounts\n// Parse the provided input and return the answer as a string.\nclass Solution {\n    public String solve(String input) {\n        return \"\";\n    }\n}\n","cpp":"// Unioned Accounts\n// Parse the provided input and return the answer as a string.\n#include <string>\nusing namespace std;\n\nclass Solution {\npublic:\n    string solve(const string& input) {\n        return \"\";\n    }\n};\n","go":"// Unioned Accounts\n// Parse stdin and print the answer.\npackage main\n\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"os\"\n)\n\nfunc Solve(input string) string {\n\treturn \"\"\n}\n\nfunc main() {\n\tinput, err := io.ReadAll(os.Stdin)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Print(Solve(string(input)))\n}\n","rust":"// Unioned Accounts\n// Parse the provided input and return the answer as a string.\npub fn solve(input: &str) -> String {\n    String::new()\n}\n","zig":"// Unioned Accounts\n// Parse stdin and print the answer.\nconst std = @import(\"std\");\n\nfn solve(input: []const u8, allocator: std.mem.Allocator) ![]u8 {\n    _ = allocator;\n    _ = input;\n    return \"\";\n}\n\npub fn main() !void {\n    var gpa = std.heap.GeneralPurposeAllocator(.{}){};\n    defer _ = gpa.deinit();\n    const allocator = gpa.allocator();\n\n    var stdin = std.io.getStdIn().reader();\n    const input = try stdin.readAllAlloc(allocator, 1 << 20);\n    defer allocator.free(input);\n\n    const output = try solve(input, allocator);\n    defer allocator.free(output);\n\n    try std.io.getStdOut().writer().print(\"{s}\", .{output});\n}\n","csharp":"// Unioned Accounts\n// Parse stdin and print the answer.\nusing System;\n\nclass Program\n{\n    static string Solve(string input)\n    {\n        return \"\";\n    }\n\n    static void Main()\n    {\n        var input = Console.In.ReadToEnd();\n        Console.Write(Solve(input));\n    }\n}\n","c":"// Unioned Accounts\n// Parse stdin and print the answer.\n#include <stdio.h>\n#include <stdlib.h>\n\nchar* solve(const char* input) {\n    (void)input;\n    char* out = malloc(1);\n    out[0] = ''\\0'';\n    return out;\n}\n\nint main(void) {\n    char* input = NULL;\n    size_t cap = 0;\n    size_t len = 0;\n\n    int ch;\n    while ((ch = getchar()) != EOF) {\n        if (len + 1 >= cap) {\n            cap = cap == 0 ? 256 : cap * 2;\n            input = realloc(input, cap);\n        }\n        input[len++] = (char)ch;\n    }\n    if (!input) {\n        input = malloc(1);\n    }\n    input[len] = ''\\0'';\n\n    char* out = solve(input);\n    fputs(out, stdout);\n\n    free(out);\n    free(input);\n    return 0;\n}\n","ruby":"# Unioned Accounts\n# Parse stdin and print the answer.\ndef solve(input)\n  \"\"\nend\n\ninput = STDIN.read\nprint solve(input)\n","php":"<?php\n// Unioned Accounts\n// Parse stdin and print the answer.\nfunction solve(string $input): string {\n    return \"\";\n}\n\n$input = stream_get_contents(STDIN);\necho solve($input);\n"},"acceptanceRate":48,"estimatedMinutes":34,"attempts":13820,"popularity":83,"createdAt":"2026-02-20","progress":"in-progress","testCases":[{"name":"Case 1","input":"Rin r@cf.dev rin@cf.dev\nKai k@cf.dev\nRin rin@cf.dev ops@cf.dev","expectedOutput":"Rin ops@cf.dev r@cf.dev rin@cf.dev\nKai k@cf.dev"}],"languageVersions":{"typescript":"5.8","javascript":"node22","python":"3.12","java":"21","cpp":"gnu++23","go":"1.24","rust":"1.80","zig":"0.13","csharp":"12.0","c":"c23","ruby":"3.3","php":"8.3"}}'::jsonb,
    '2026-02-20T00:00:00Z'::timestamptz,
    now()
)
on conflict (slug) do update set
    title = excluded.title,
    statement_md = excluded.statement_md,
    difficulty = excluded.difficulty,
    time_limit_ms = excluded.time_limit_ms,
    memory_limit_kb = excluded.memory_limit_kb,
    is_active = true,
    frontend_payload = excluded.frontend_payload,
    updated_at = now();
update task_languages
set is_enabled = false
where task_id = 'd3755433-bd98-47eb-a734-904eee2713b2'::uuid
  and language not in ('typescript', 'javascript', 'python', 'java', 'cpp', 'go', 'rust', 'zig', 'csharp', 'c', 'ruby', 'php');
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '0b82d29c-6c60-4e7d-9f5f-98751d4cba9f',
    'd3755433-bd98-47eb-a734-904eee2713b2'::uuid,
    'typescript',
    '5.8',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'cde945fc-a169-4af4-9b4f-df3104fde9fe',
    'd3755433-bd98-47eb-a734-904eee2713b2'::uuid,
    'javascript',
    'node22',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'fc9d9355-1366-4ad0-8ea1-e20f185bc535',
    'd3755433-bd98-47eb-a734-904eee2713b2'::uuid,
    'python',
    '3.12',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '4ea735cd-b53f-4280-ab55-461b81793f84',
    'd3755433-bd98-47eb-a734-904eee2713b2'::uuid,
    'java',
    '21',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'e55f9559-0f1d-4c59-9905-075391794b93',
    'd3755433-bd98-47eb-a734-904eee2713b2'::uuid,
    'cpp',
    'gnu++23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'e9fe99a0-058f-4661-9d93-41cd849fe7b4',
    'd3755433-bd98-47eb-a734-904eee2713b2'::uuid,
    'go',
    '1.24',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'a7395d39-e388-45bb-8677-17a3605de84b',
    'd3755433-bd98-47eb-a734-904eee2713b2'::uuid,
    'rust',
    '1.80',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '8e2da9e3-233a-4580-99b2-b0abf4264b61',
    'd3755433-bd98-47eb-a734-904eee2713b2'::uuid,
    'zig',
    '0.13',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '53c69eb4-222d-484b-9eda-dd5d386ba41b',
    'd3755433-bd98-47eb-a734-904eee2713b2'::uuid,
    'csharp',
    '12.0',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '42604bcd-ddb9-4bae-b1a8-27838d3e9b3a',
    'd3755433-bd98-47eb-a734-904eee2713b2'::uuid,
    'c',
    'c23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '2808321f-52d6-417d-b57d-0e246e4c9f0b',
    'd3755433-bd98-47eb-a734-904eee2713b2'::uuid,
    'ruby',
    '3.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'e04bef99-6226-4b02-871f-7ae097d25724',
    'd3755433-bd98-47eb-a734-904eee2713b2'::uuid,
    'php',
    '8.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    'afc8df14-801a-45db-ad78-1e12087e44c9',
    'd3755433-bd98-47eb-a734-904eee2713b2'::uuid,
    'sample',
    1,
    'Rin r@cf.dev rin@cf.dev
Kai k@cf.dev
Rin rin@cf.dev ops@cf.dev',
    'Rin ops@cf.dev r@cf.dev rin@cf.dev
Kai k@cf.dev',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
update task_test_cases
set is_active = false
where task_id = 'd3755433-bd98-47eb-a734-904eee2713b2'::uuid
  and kind = 'sample'
  and order_no not in (1);

-- sensor-drift
insert into tasks (
    id, slug, title, statement_md, difficulty, time_limit_ms, memory_limit_kb, is_active, frontend_payload, created_at, updated_at
) values (
    '61fb3e4e-4615-4cd6-a8ff-91c12b2219fe',
    'sensor-drift',
    'Sensor Drift',
    'A lab sensor reports integer readings every second. For each window of size k, compute the average reading.

Return the first window start index whose average is greater than or equal to the alert threshold. If no window qualifies, return -1.',
    'easy',
    2000,
    262144,
    true,
    '{"id":"challenge-016","slug":"sensor-drift","title":"Sensor Drift","difficulty":"Easy","summary":"Find the smallest moving average that crosses an alert threshold.","description":["A lab sensor reports integer readings every second. For each window of size k, compute the average reading.","Return the first window start index whose average is greater than or equal to the alert threshold. If no window qualifies, return -1."],"examples":[{"title":"Alert crossed","input":"readings = [2, 3, 8, 4, 6], k = 3, threshold = 5","output":"1"},{"title":"No alert","input":"readings = [1, 2, 3], k = 2, threshold = 5","output":"-1"}],"constraints":["1 <= k <= readings.length <= 100,000","-10,000 <= readings[i] <= 10,000","threshold may be fractional."],"notes":["Compare sums against threshold * k to avoid repeated division."],"tags":["Sliding Window","Math","Array"],"category":"Warmup","kind":"algorithmic","supportedLanguages":["typescript","javascript","python","java","cpp","go","rust","zig","csharp","c","ruby","php"],"starterCodeByLanguage":{"typescript":"// Sensor Drift\n// Parse the provided input and return the answer as a string.\nexport function solve(input: string): string {\n  return \"\";\n}\n","javascript":"// Sensor Drift\n// Parse the provided input and return the answer as a string.\nfunction solve(input) {\n  return \"\";\n}\n\nmodule.exports = { solve };\n","python":"# Sensor Drift\n# Parse the provided input and return the answer as a string.\ndef solve(input_data: str) -> str:\n    return \"\"\n","java":"// Sensor Drift\n// Parse the provided input and return the answer as a string.\nclass Solution {\n    public String solve(String input) {\n        return \"\";\n    }\n}\n","cpp":"// Sensor Drift\n// Parse the provided input and return the answer as a string.\n#include <string>\nusing namespace std;\n\nclass Solution {\npublic:\n    string solve(const string& input) {\n        return \"\";\n    }\n};\n","go":"// Sensor Drift\n// Parse stdin and print the answer.\npackage main\n\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"os\"\n)\n\nfunc Solve(input string) string {\n\treturn \"\"\n}\n\nfunc main() {\n\tinput, err := io.ReadAll(os.Stdin)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Print(Solve(string(input)))\n}\n","rust":"// Sensor Drift\n// Parse the provided input and return the answer as a string.\npub fn solve(input: &str) -> String {\n    String::new()\n}\n","zig":"// Sensor Drift\n// Parse stdin and print the answer.\nconst std = @import(\"std\");\n\nfn solve(input: []const u8, allocator: std.mem.Allocator) ![]u8 {\n    _ = allocator;\n    _ = input;\n    return \"\";\n}\n\npub fn main() !void {\n    var gpa = std.heap.GeneralPurposeAllocator(.{}){};\n    defer _ = gpa.deinit();\n    const allocator = gpa.allocator();\n\n    var stdin = std.io.getStdIn().reader();\n    const input = try stdin.readAllAlloc(allocator, 1 << 20);\n    defer allocator.free(input);\n\n    const output = try solve(input, allocator);\n    defer allocator.free(output);\n\n    try std.io.getStdOut().writer().print(\"{s}\", .{output});\n}\n","csharp":"// Sensor Drift\n// Parse stdin and print the answer.\nusing System;\n\nclass Program\n{\n    static string Solve(string input)\n    {\n        return \"\";\n    }\n\n    static void Main()\n    {\n        var input = Console.In.ReadToEnd();\n        Console.Write(Solve(input));\n    }\n}\n","c":"// Sensor Drift\n// Parse stdin and print the answer.\n#include <stdio.h>\n#include <stdlib.h>\n\nchar* solve(const char* input) {\n    (void)input;\n    char* out = malloc(1);\n    out[0] = ''\\0'';\n    return out;\n}\n\nint main(void) {\n    char* input = NULL;\n    size_t cap = 0;\n    size_t len = 0;\n\n    int ch;\n    while ((ch = getchar()) != EOF) {\n        if (len + 1 >= cap) {\n            cap = cap == 0 ? 256 : cap * 2;\n            input = realloc(input, cap);\n        }\n        input[len++] = (char)ch;\n    }\n    if (!input) {\n        input = malloc(1);\n    }\n    input[len] = ''\\0'';\n\n    char* out = solve(input);\n    fputs(out, stdout);\n\n    free(out);\n    free(input);\n    return 0;\n}\n","ruby":"# Sensor Drift\n# Parse stdin and print the answer.\ndef solve(input)\n  \"\"\nend\n\ninput = STDIN.read\nprint solve(input)\n","php":"<?php\n// Sensor Drift\n// Parse stdin and print the answer.\nfunction solve(string $input): string {\n    return \"\";\n}\n\n$input = stream_get_contents(STDIN);\necho solve($input);\n"},"acceptanceRate":76,"estimatedMinutes":15,"attempts":10220,"popularity":79,"createdAt":"2026-04-01","progress":"not-started","testCases":[{"name":"Case 1","input":"[2, 3, 8, 4, 6]\n3\n5","expectedOutput":"1"},{"name":"Case 2","input":"[1, 2, 3]\n2\n5","expectedOutput":"-1"}],"languageVersions":{"typescript":"5.8","javascript":"node22","python":"3.12","java":"21","cpp":"gnu++23","go":"1.24","rust":"1.80","zig":"0.13","csharp":"12.0","c":"c23","ruby":"3.3","php":"8.3"}}'::jsonb,
    '2026-04-01T00:00:00Z'::timestamptz,
    now()
)
on conflict (slug) do update set
    title = excluded.title,
    statement_md = excluded.statement_md,
    difficulty = excluded.difficulty,
    time_limit_ms = excluded.time_limit_ms,
    memory_limit_kb = excluded.memory_limit_kb,
    is_active = true,
    frontend_payload = excluded.frontend_payload,
    updated_at = now();
update task_languages
set is_enabled = false
where task_id = '61fb3e4e-4615-4cd6-a8ff-91c12b2219fe'::uuid
  and language not in ('typescript', 'javascript', 'python', 'java', 'cpp', 'go', 'rust', 'zig', 'csharp', 'c', 'ruby', 'php');
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '8cb3db0a-08dc-4429-8129-c233afa09b5b',
    '61fb3e4e-4615-4cd6-a8ff-91c12b2219fe'::uuid,
    'typescript',
    '5.8',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '6407f790-7fa9-423e-8bbe-104b11b5e73d',
    '61fb3e4e-4615-4cd6-a8ff-91c12b2219fe'::uuid,
    'javascript',
    'node22',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'f54506fd-eb0e-4f8e-9a3d-d8cc3ab9b544',
    '61fb3e4e-4615-4cd6-a8ff-91c12b2219fe'::uuid,
    'python',
    '3.12',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'b5180926-2c14-4be4-b1f7-438049d3a169',
    '61fb3e4e-4615-4cd6-a8ff-91c12b2219fe'::uuid,
    'java',
    '21',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'cec7f8cb-46ee-484c-852d-6d8db3cecb78',
    '61fb3e4e-4615-4cd6-a8ff-91c12b2219fe'::uuid,
    'cpp',
    'gnu++23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '681b3425-805b-4a75-988b-47cef0685d84',
    '61fb3e4e-4615-4cd6-a8ff-91c12b2219fe'::uuid,
    'go',
    '1.24',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '0ea16646-305d-4567-afc6-e4d15f7d5957',
    '61fb3e4e-4615-4cd6-a8ff-91c12b2219fe'::uuid,
    'rust',
    '1.80',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '237a5314-30a5-407e-b00c-6fb06fdf75f8',
    '61fb3e4e-4615-4cd6-a8ff-91c12b2219fe'::uuid,
    'zig',
    '0.13',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '070f7b33-8139-4cc8-8366-13b54a59f5bf',
    '61fb3e4e-4615-4cd6-a8ff-91c12b2219fe'::uuid,
    'csharp',
    '12.0',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '11597431-342b-47b4-ae27-7c68c1c73f54',
    '61fb3e4e-4615-4cd6-a8ff-91c12b2219fe'::uuid,
    'c',
    'c23',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '0750ea01-7904-4121-9647-61f9482ce5a1',
    '61fb3e4e-4615-4cd6-a8ff-91c12b2219fe'::uuid,
    'ruby',
    '3.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    '9b01d245-8ce8-4e11-8cbb-598f6f74e55e',
    '61fb3e4e-4615-4cd6-a8ff-91c12b2219fe'::uuid,
    'php',
    '8.3',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    'f5a8806d-8f85-4261-8ed0-4c367c58f27d',
    '61fb3e4e-4615-4cd6-a8ff-91c12b2219fe'::uuid,
    'sample',
    1,
    '[2, 3, 8, 4, 6]
3
5',
    '1',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    '7988496d-d31c-448e-94b2-a46b88011b59',
    '61fb3e4e-4615-4cd6-a8ff-91c12b2219fe'::uuid,
    'sample',
    2,
    '[1, 2, 3]
2
5',
    '-1',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
update task_test_cases
set is_active = false
where task_id = '61fb3e4e-4615-4cd6-a8ff-91c12b2219fe'::uuid
  and kind = 'sample'
  and order_no not in (1, 2);

-- retention-cohorts
insert into tasks (
    id, slug, title, statement_md, difficulty, time_limit_ms, memory_limit_kb, is_active, frontend_payload, created_at, updated_at
) values (
    '9266b83a-0903-4484-bea4-c2e1331bb974',
    'retention-cohorts',
    'Retention Cohorts',
    'A product analytics table stores one row per user activity event. You need to measure how many users from each signup date returned exactly seven days later.

Return signup_date, cohort_size, retained_users, and retention_rate ordered by signup_date ascending. The retention rate should be a decimal rounded to two places.',
    'medium',
    3000,
    262144,
    true,
    '{"id":"challenge-017","slug":"retention-cohorts","title":"Retention Cohorts","difficulty":"Medium","summary":"Write a SQL query that reports day-seven user retention by signup date.","description":["A product analytics table stores one row per user activity event. You need to measure how many users from each signup date returned exactly seven days later.","Return signup_date, cohort_size, retained_users, and retention_rate ordered by signup_date ascending. The retention rate should be a decimal rounded to two places."],"examples":[{"title":"Small activity sample","input":"events(user_id, event_date, signup_date)\n(1, ''2026-04-01'', ''2026-04-01'')\n(1, ''2026-04-08'', ''2026-04-01'')\n(2, ''2026-04-01'', ''2026-04-01'')","output":"2026-04-01 | 2 | 1 | 0.50","explanation":"Two users signed up on April 1. Only one of them had activity exactly seven days later."}],"constraints":["The table name is events.","event_date and signup_date are DATE columns.","Each user may have multiple activity events on the same date.","Return one row per signup date."],"notes":["Count distinct retained users so duplicate activity does not inflate the rate."],"tags":["Database","Hash Table"],"category":"SQL","kind":"sql","supportedLanguages":["sql"],"starterCodeByLanguage":{"sql":"-- Retention Cohorts\n-- Write a query that returns the requested result.\nSELECT 1;\n"},"acceptanceRate":57,"estimatedMinutes":26,"attempts":7180,"popularity":74,"createdAt":"2026-04-06","progress":"not-started","testCases":[{"name":"Case 1","input":"events\n1 2026-04-01 2026-04-01\n1 2026-04-08 2026-04-01\n2 2026-04-01 2026-04-01","expectedOutput":"2026-04-01 | 2 | 1 | 0.50"}],"languageVersions":{"sql":"ansi"}}'::jsonb,
    '2026-04-06T00:00:00Z'::timestamptz,
    now()
)
on conflict (slug) do update set
    title = excluded.title,
    statement_md = excluded.statement_md,
    difficulty = excluded.difficulty,
    time_limit_ms = excluded.time_limit_ms,
    memory_limit_kb = excluded.memory_limit_kb,
    is_active = true,
    frontend_payload = excluded.frontend_payload,
    updated_at = now();
update task_languages
set is_enabled = false
where task_id = '9266b83a-0903-4484-bea4-c2e1331bb974'::uuid
  and language not in ('sql');
insert into task_languages (
    id, task_id, language, language_version, is_enabled, created_at
) values (
    'fe37bd25-96cf-4cee-b0e8-2c06031f38ef',
    '9266b83a-0903-4484-bea4-c2e1331bb974'::uuid,
    'sql',
    'ansi',
    true,
    now()
)
on conflict (task_id, language, language_version) do update set
    is_enabled = true;
insert into task_test_cases (
    id, task_id, kind, order_no, input_blob_ref, expected_blob_ref, weight, is_active, created_at
) values (
    '2038445e-7141-4e42-8e17-5c1866eb7b7a',
    '9266b83a-0903-4484-bea4-c2e1331bb974'::uuid,
    'sample',
    1,
    'events
1 2026-04-01 2026-04-01
1 2026-04-08 2026-04-01
2 2026-04-01 2026-04-01',
    '2026-04-01 | 2 | 1 | 0.50',
    1,
    true,
    now()
)
on conflict (task_id, order_no) do update set
    kind = excluded.kind,
    input_blob_ref = excluded.input_blob_ref,
    expected_blob_ref = excluded.expected_blob_ref,
    weight = excluded.weight,
    is_active = true;
update task_test_cases
set is_active = false
where task_id = '9266b83a-0903-4484-bea4-c2e1331bb974'::uuid
  and kind = 'sample'
  and order_no not in (1);
commit;
