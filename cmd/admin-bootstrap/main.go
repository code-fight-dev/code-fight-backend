package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"code-fight-backend/internal/administration"
	"code-fight-backend/internal/database"

	"github.com/google/uuid"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	userID := flag.String("user-id", "", "Existing account UUID to make the first admin")
	reason := flag.String("reason", "", "Required audit reason")
	flag.Parse()
	id, err := uuid.Parse(*userID)
	if err != nil || id == uuid.Nil || strings.TrimSpace(*reason) == "" || flag.NArg() != 0 {
		return fmt.Errorf("usage: admin-bootstrap -user-id UUID -reason 'Initial administrator setup'")
	}
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL is required; apply backend migrations before bootstrap")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := database.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()
	user, err := administration.NewService(db).BootstrapAdmin(ctx, id, *reason, uuid.NewString())
	if err != nil {
		return err
	}
	fmt.Printf("First admin created: %s (%s), role version %d\n", user.Username, user.ID, user.RoleVersion)
	return nil
}
