package notifications

import (
	"context"
	"log/slog"
	"time"

	"code-fight-backend/internal/realtime"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Start(ctx context.Context, db *pgxpool.Pool, broker *realtime.Broker, logger *slog.Logger, tournamentReadyInterval string) {
	go Listen(ctx, db, broker, logger)
	go RunScheduled(ctx, db, logger, tournamentReadyInterval)
}

// Listen relays PostgreSQL notifications after their originating transactions commit.
func Listen(ctx context.Context, db *pgxpool.Pool, broker *realtime.Broker, logger *slog.Logger) {
	config := db.Config().ConnConfig
	for ctx.Err() == nil {
		if err := relay(ctx, config, broker); err != nil && ctx.Err() == nil {
			logger.Warn("notification stream listener disconnected", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

func relay(ctx context.Context, config *pgx.ConnConfig, broker *realtime.Broker) error {
	conn, err := pgx.ConnectConfig(ctx, config.Copy())
	if err != nil {
		return err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = conn.Close(closeCtx)
	}()
	if _, err := conn.Exec(ctx, "listen codefight_notifications"); err != nil {
		return err
	}
	for {
		notice, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		if recipient, err := uuid.Parse(notice.Payload); err == nil {
			broker.Publish(recipient, ChangedEvent, nil)
		}
	}
}

func RunScheduled(ctx context.Context, db *pgxpool.Pool, logger *slog.Logger, tournamentReadyInterval string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		runCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
		if err := deliverScheduled(runCtx, db, tournamentReadyInterval); err != nil && ctx.Err() == nil {
			logger.Warn("scheduled notification delivery failed", "error", err)
		}
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func deliverScheduled(ctx context.Context, db *pgxpool.Pool, tournamentReadyInterval string) error {
	if _, err := db.Exec(ctx, `update notifications set deliver_at=null where id in (
		select id from notifications where deliver_at <= now()
		order by deliver_at limit 100 for update skip locked
	)`); err != nil {
		return err
	}
	_, err := db.Exec(ctx, `
insert into notifications (user_id,kind,title,message,href,deadline_at,event_key)
select p.user_id,'tournament.ready_reminder','Confirm match readiness',
       t.title || ' · Your readiness deadline is approaching.',
       '/arena/match/' || m.id,m.created_at+$1::interval,m.id::text
from matches m
join tournament_fixtures f on f.match_id=m.id
join tournaments t on t.id=f.tournament_id
cross join lateral (values (m.player1_id,m.player1_ready),(m.player2_id,m.player2_ready)) p(user_id,ready)
where m.status='pending' and not p.ready
  and m.created_at+$1::interval > now()
  and m.created_at+$1::interval <= now()+interval '3 minutes'
on conflict (user_id,kind,event_key) do nothing`, tournamentReadyInterval)
	return err
}
