package matches

import (
	"bytes"
	"context"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PreparePlayersForMatch holds the same player locks as queue admission and
// removes the participants from matchmaking in the match creation transaction.
func PreparePlayersForMatch(ctx context.Context, tx pgx.Tx, players ...uuid.UUID) error {
	if err := lockAvailablePlayers(ctx, tx, players); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, removeQueuedPlayersQuery, players)
	return err
}

func lockAvailablePlayers(ctx context.Context, tx pgx.Tx, players []uuid.UUID) error {
	slices.SortFunc(players, func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })
	for _, player := range players {
		if _, err := tx.Exec(ctx,
			"select pg_advisory_xact_lock(hashtextextended('match-player:' || $1::text, 0))", player.String(),
		); err != nil {
			return err
		}
	}
	var active bool
	if err := tx.QueryRow(ctx, `select exists (select 1 from matches
		where status in ('pending','running') and (player1_id = any($1) or player2_id = any($1)))`, players).Scan(&active); err != nil {
		return err
	}
	if active {
		return ErrPlayerAlreadyInMatch
	}
	return nil
}
