# Single-elimination tournament API

Publish an organizer draft via `POST /api/organizer/tournaments/{id}/status`
with `status: "published"`, `format: "single_elimination"`, one `taskSlug`,
`expectedConfigVersion`, and the three RFC 3339 schedule fields
`registrationClosesAt`, `startsAt`, and `endsAt`. Use the existing join endpoint.

At the start, the scheduler draws 2–16 registered players at random and stores
all rounds. First-round byes advance immediately. With fewer than two players
the tournament closes without a champion.

`GET /api/tournaments/{id}` includes `bracket.slots` ordered by one-based
`round` and `position`. Each slot exposes both player IDs and usernames,
`winnerId`, `fixtureId`, `matchId`, and its destination `nextSlotId` and
`nextSlotSide` (1 or 2). Empty sides and the final slot's destination are null.
Before the draw the slots array is empty.

Render slots by status. A `waiting` slot depends on an earlier round. A
`bye` identifies an automatic advance. For `ready` and `replay_required`,
the organizer can start a duel via the existing fixture scheduling endpoint.
A draw or cancellation requires a replay. The existing match flow handles
`pending` and `running`. The scheduler advances each decisive winner and
creates the next fixture once both players are known.

After the final, `bracket.championId` identifies the champion. Places are
determined by the round reached, with ties for elimination in the same round.
History stores the final place. Contest rating includes players who played
a decisive duel. At the deadline the scheduler waits for admitted submissions
to settle. If the final remains undecided, the tournament closes with a null
champion, unresolved slots become `closed`, and ratings stay unchanged.
