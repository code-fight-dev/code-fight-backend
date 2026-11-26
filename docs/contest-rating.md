# Contest rating

Finished fixtures award three points for a win and one for a draw. Standings
rank by points and then wins, both descending. Players tied on both share a
dense rank. Username and user ID determine display order within a tie.
Unscheduled and cancelled fixtures award zero points.

Contest ratings start at 1000 and are stored separately from duel ratings.
At completion, each player with at least one finished fixture is compared
with every other eligible player by rank. A better rank scores 1, a tie
scores 0.5 and a worse rank scores 0. For ratings `Ri` and `Rj`, the expected
score is `1 / (1 + 10^((Rj - Ri)/400))`. The change is 32 times the average
of actual minus expected scores, rounded to the nearest integer. All
comparisons use the pre-event ratings. At least two eligible players are
required. Previously completed events remain unrated.

The tournament board adds `rank`, `contestRatingBefore`, and
`contestRatingAfter` to each standing. Participation history returns the
same two rating values for each event. Rating values are `null` when the
player received no rating change record. The UI can subtract the before
value from the after value to display the event delta.
