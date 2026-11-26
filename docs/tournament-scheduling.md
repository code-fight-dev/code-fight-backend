# Tournament deadlines

Schedule fields are RFC 3339 timestamps and are returned in UTC. A match's
submission cutoff is the earlier of its own expiry and the tournament's
`endsAt`. Backend admits submissions strictly before that cutoff and rejects
later submissions with HTTP 409.

An admitted submission remains eligible even when Judge returns its verdict
after `endsAt`. The tournament stays `running` while eligible executions or
match updates are pending. The scheduler then settles the results, cancels
any remaining unfinished fixtures, and marks the tournament `completed`.
Final standings and participation history include the eligible verdicts.
