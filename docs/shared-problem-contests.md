# Shared problem-set contests

Create an organizer draft with `POST /api/organizer/tournaments`. Then
publish it with `POST /api/organizer/tournaments/{id}/status`.

```json
{
  "status": "published",
  "format": "shared_problem_set",
  "taskSlugs": ["mirror-log", "signal-stabilizer"],
  "registrationClosesAt": "2026-12-01T18:55:00Z",
  "startsAt": "2026-12-01T19:00:00Z",
  "endsAt": "2026-12-01T20:00:00Z",
  "expectedConfigVersion": 1
}
```

Publication requires an ordered list of 1–20 distinct active tasks, each
with an enabled language. All dates are RFC 3339 and responses use UTC.
Registration closes at or before the start. Up to 16 participants can
register with `POST /api/tournaments/{id}/join`.

`GET /api/tournaments/{id}` returns `tournament.format` and `problems`.
The problem list is empty until `startsAt`. After the start it contains
`taskId`, `slug`, `title`, `difficulty`, and one-based `position`. Fetch a
problem and the signed-in user's attempts at `GET /api/tasks/id/{taskId}`.
Filter `submissionHistory` by `tournamentId` for the current contest.

Send `language`, `language_version`, and `source_code` as JSON to
`POST /api/tournaments/{id}/problems/{taskId}/submissions`. The authenticated
participant receives HTTP 201 and the existing submission response with
`tournament_id`. Follow the verdict via `GET /api/submissions/{submissionId}`
or the existing submission events. An unassigned problem returns 404, an
unregistered user receives 403, and a closed submission window returns 409.
The server checks `startsAt <= admittedAt < endsAt` at admission.

Standings rank by solved problems descending, then penalty seconds
ascending. For each solved problem, take the server admission time of the
first accepted solution in whole seconds since the start, plus 1,200 seconds
for each earlier judged, non-accepted attempt on that problem. Compilation
errors and infrastructure failures add zero penalty. Equal scores share the
same dense rank, so the rank after two users tied for first is second.
The board returns `rank`, `solved`, `penaltySeconds`, and `judgedSubmissions`.

At the deadline the scheduler waits for admitted submissions to settle,
then saves final standings and contest rating changes in one transaction.
Players with at least one judged result are eligible for rating. At least two
eligible players are required. `GET /api/tournaments/history` includes the
participant's final rank, solved count, penalty seconds and rating change.
Contest submissions have their own scope in Judge metrics and are excluded
from the practice analytics API.
