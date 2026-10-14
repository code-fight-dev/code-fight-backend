package foundation_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"code-fight-backend/internal/access"
	"code-fight-backend/internal/administration"
	"code-fight-backend/internal/managementtx"
	"code-fight-backend/internal/tournament"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestF1ForeignDraftDoesNotWaitForRowLock(t *testing.T) {
	f := newFixture(t)
	f.setupRoles(t)
	draft, err := f.tournaments.CreateDraft(f.ctx, f.users[1].ID,
		tournament.CreateInput{Slug: "locked-cup", Title: "Locked Cup"}, uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	gate, err := f.db.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer gate.Rollback(f.ctx)
	if _, err := gate.Exec(f.ctx, `select id from tournaments where id=$1 for update`, draft.ID); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(f.ctx, 5*time.Second)
	defer cancel()
	_, err = f.tournaments.UpdateDraft(ctx, f.users[2].ID, draft.ID,
		tournament.UpdateInput{Title: stringPtr("Foreign edit"), ExpectedConfigVersion: 1}, uuid.NewString())
	if !errors.Is(err, access.ErrNotFound) {
		t.Fatalf("foreign edit must return not found while the row remains locked: %v", err)
	}
	if _, err := f.admin.ChangeRole(ctx, f.users[0].ID, f.users[2].ID,
		administration.ChangeRoleInput{Role: access.User, ExpectedRoleVersion: 2, Reason: "Revoke foreign editor"}, uuid.NewString()); err != nil {
		t.Fatalf("rejected edit retained the actor lock: %v", err)
	}
}

func TestF1BlockedEditReleasesTransaction(t *testing.T) {
	for _, mode := range []string{"lock timeout", "request canceled"} {
		t.Run(mode, func(t *testing.T) {
			f := newFixture(t)
			f.setupRoles(t)
			draft, err := f.tournaments.CreateDraft(f.ctx, f.users[1].ID,
				tournament.CreateInput{Slug: "blocked-cup", Title: "Original title"}, uuid.NewString())
			if err != nil {
				t.Fatal(err)
			}
			gate, err := f.db.Begin(f.ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer gate.Rollback(f.ctx)
			if _, err := gate.Exec(f.ctx, `select id from tournaments where id=$1 for update`, draft.ID); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(f.ctx)
			defer cancel()
			result := make(chan error, 1)
			go func() {
				_, err := f.tournaments.UpdateDraft(ctx, f.users[1].ID, draft.ID,
					tournament.UpdateInput{Title: stringPtr("Must not persist"), ExpectedConfigVersion: 1}, uuid.NewString())
				result <- err
			}()
			waitForBlocked(t, f, 1)
			if mode == "request canceled" {
				cancel()
			}
			select {
			case err = <-result:
			case <-time.After(5 * time.Second):
				t.Fatal("blocked edit did not finish within the lock-wait bound")
			}
			if mode == "request canceled" {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("canceled edit error = %v", err)
				}
			} else {
				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
					t.Fatalf("expected PostgreSQL lock timeout, got %v", err)
				}
			}

			revokeCtx, revokeCancel := context.WithTimeout(f.ctx, 5*time.Second)
			defer revokeCancel()
			if _, err := f.admin.ChangeRole(revokeCtx, f.users[0].ID, f.users[1].ID,
				administration.ChangeRoleInput{Role: access.User, ExpectedRoleVersion: 2, Reason: "Revoke blocked editor"}, uuid.NewString()); err != nil {
				t.Fatalf("failed edit retained its transaction or actor lock: %v", err)
			}
			stored, err := f.tournaments.GetDraft(f.ctx, f.users[0].ID, draft.ID)
			if err != nil || stored.Title != draft.Title || stored.ConfigVersion != 1 {
				t.Fatalf("failed edit changed the draft: %+v, %v", stored, err)
			}
			if f.count(t, `select count(*) from audit_events where tournament_id=$1`, draft.ID) != 1 {
				t.Fatal("failed edit created an audit event")
			}
		})
	}
}

func TestF1ManagementTimeoutsStayTransactionLocal(t *testing.T) {
	db, ctx := isolatedDB(t)
	conn, err := db.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	var originalLock, originalStatement string
	const settings = `select current_setting('lock_timeout'), current_setting('statement_timeout')`
	if err := conn.QueryRow(ctx, settings).Scan(&originalLock, &originalStatement); err != nil {
		t.Fatal(err)
	}
	opCtx, tx, cleanup, err := managementtx.Begin(ctx, conn)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var lock, statement string
	if err := tx.QueryRow(opCtx, settings).Scan(&lock, &statement); err != nil || lock != "2s" || statement != "10s" {
		t.Fatalf("management limits = %q/%q: %v", lock, statement, err)
	}
	if err := tx.Commit(opCtx); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(ctx, settings).Scan(&lock, &statement); err != nil || lock != originalLock || statement != originalStatement {
		t.Fatalf("management limits leaked into the session: %q/%q: %v", lock, statement, err)
	}
}

func TestF1HTTPRejectsNULWithoutMutations(t *testing.T) {
	f := newFixture(t)
	f.setupRoles(t)
	draft, err := f.tournaments.CreateDraft(f.ctx, f.users[1].ID,
		tournament.CreateInput{Slug: "valid-cup", Title: "Original title", Description: "Original description"}, uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	before := f.count(t, `select count(*) from audit_events`)
	path := "/api/organizer/tournaments/" + draft.ID.String()
	for _, test := range []struct {
		name, method, path, body string
		actor                    int
	}{
		{"create slug", "POST", "/api/organizer/tournaments", `{"slug":"invalid\u0000slug","title":"Title"}`, 1},
		{"create title", "POST", "/api/organizer/tournaments", `{"slug":"new-cup","title":"invalid\u0000title"}`, 1},
		{"create description", "POST", "/api/organizer/tournaments", `{"slug":"new-cup","title":"Title","description":"invalid\u0000description"}`, 1},
		{"edit slug", "PATCH", path, `{"slug":"invalid\u0000slug","expectedConfigVersion":1}`, 1},
		{"edit title", "PATCH", path, `{"title":"invalid\u0000title","expectedConfigVersion":1}`, 1},
		{"edit description", "PATCH", path, `{"description":"invalid\u0000description","expectedConfigVersion":1}`, 1},
		{"role reason", "PATCH", "/api/admin/users/" + f.users[1].ID.String() + "/role",
			`{"role":"user","expectedRoleVersion":2,"reason":"invalid\u0000reason"}`, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := f.request(t, test.actor, test.method, test.path, test.body, 400)
			var envelope struct {
				Error struct{ Code string } `json:"error"`
			}
			if err := json.Unmarshal(body, &envelope); err != nil || envelope.Error.Code != "invalid_request" {
				t.Fatalf("unexpected validation response: %s", body)
			}
		})
	}
	stored, err := f.tournaments.GetDraft(f.ctx, f.users[1].ID, draft.ID)
	if err != nil || stored.Title != draft.Title || stored.Description != draft.Description || stored.ConfigVersion != 1 {
		t.Fatalf("invalid input changed the draft: %+v, %v", stored, err)
	}
	user, err := f.auth.CurrentUser(f.ctx, f.tokens[1])
	if err != nil || user.Role != access.Organizer || user.RoleVersion != 2 {
		t.Fatalf("invalid reason changed the role: %+v, %v", user, err)
	}
	if f.count(t, `select count(*) from audit_events`) != before || f.count(t, `select count(*) from tournaments`) != 1 {
		t.Fatal("invalid input committed a mutation or audit event")
	}
}
