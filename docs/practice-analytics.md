# Practice analytics API

Both endpoints use the signed-in user's session. Practice includes submissions
without a match. Code runs, duel submissions and tournament submissions are
excluded.

`GET /api/practice/summary` accepts `from` and `to` as inclusive `YYYY-MM-DD`
dates and `timeZone` as an IANA time zone. Defaults are UTC and the last 365
calendar days including today. The range may contain up to 366 days and must
end by today in the selected time zone.

- `totals` counts all practice submissions and distinct attempted and solved
  tasks. A task is solved after at least one `accepted` verdict.
- `activity` echoes the range and time zone. `days` contains one row per date
  with `date`, `submissions` and `firstSolves`, including empty days. Both
  counts use submission time. A first solve is the earliest accepted
  submission across all practice history, even when the response window is
  shorter.
- `streaks` counts consecutive local days with submissions across all history.
  A streak ending yesterday is current until the end of today.
- `byDifficulty`, `byCategory` and `byTag` list `key`, `attemptedTasks`,
  `solvedTasks` and `availableTasks`. Availability counts active catalog tasks.
  Attempts and solves include retired tasks. A task with multiple tags
  contributes to each tag.
- `byLanguage` lists `language`, `submissions`, `acceptedSubmissions`,
  `attemptedTasks` and `solvedTasks`. Attempted and solved counts are distinct
  per language, so a task may appear in more than one language breakdown.

`GET /api/practice/submissions` accepts optional `language`, `verdict`, `limit`
and `cursor`. It returns newest first with stable ordering for submissions
sharing a timestamp. Limit defaults to 20 and allows 1–100. Send `nextCursor`
as `cursor` with the same filters to load another page. Each item includes
`submissionId`, `taskSlug`, `taskTitle`, `difficulty`, `language`,
`languageVersion`, `status`, `verdict` and `submittedAt`. A pending verdict is
`null`. The response omits source code.
