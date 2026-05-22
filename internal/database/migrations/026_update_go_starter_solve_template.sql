begin;

update tasks
set
    frontend_payload = jsonb_set(
        frontend_payload,
        '{starterCodeByLanguage,go}',
        to_jsonb(
            format(
                '// %s
// Parse stdin and return the answer as a string.
package main

import (
	"fmt"
	"io"
	"os"
)

func Solve(input string) string {
	return ""
}
',
                title
            )
        ),
        true
    ),
    updated_at = now()
where frontend_payload ? 'starterCodeByLanguage'
  and frontend_payload->'starterCodeByLanguage' ? 'go';

commit;
