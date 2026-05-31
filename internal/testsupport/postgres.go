package testsupport

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"code-fight-backend/internal/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RequirePostgres(t *testing.T) (*pgxpool.Pool, context.Context) {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("BACKEND_TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("set BACKEND_TEST_DATABASE_URL or TEST_DATABASE_URL to run Postgres integration tests")
	}
	if !strings.Contains(strings.ToLower(dsn), "test") {
		t.Fatalf("refusing to run integration tests against a non-test database URL: %s", dsn)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect test postgres: %v", err)
	}
	t.Cleanup(db.Close)

	if err := db.Ping(ctx); err != nil {
		t.Fatalf("ping test postgres: %v", err)
	}
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate test postgres: %v", err)
	}

	return db, ctx
}
