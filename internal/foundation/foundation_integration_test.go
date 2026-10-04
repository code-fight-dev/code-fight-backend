package foundation_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"code-fight-backend/internal/access"
	"code-fight-backend/internal/administration"
	"code-fight-backend/internal/auth"
	"code-fight-backend/internal/config"
	"code-fight-backend/internal/database"
	httpserver "code-fight-backend/internal/http"
	"code-fight-backend/internal/leaderboard"
	"code-fight-backend/internal/matches"
	"code-fight-backend/internal/realtime"
	"code-fight-backend/internal/submissions"
	"code-fight-backend/internal/tasks"
	"code-fight-backend/internal/tournament"
	"code-fight-backend/internal/users"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func isolatedDB(t *testing.T) (*pgxpool.Pool, context.Context) {
	t.Helper()
	dsn := os.Getenv("BACKEND_TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("TEST_DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("set BACKEND_TEST_DATABASE_URL for PostgreSQL foundation tests (requires CREATEDB)")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid test database configuration")
	}
	if !strings.Contains(strings.ToLower(cfg.ConnConfig.Database), "test") {
		t.Fatal("foundation tests require a database name containing test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	control, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	name := "codefight_f1_" + strings.ReplaceAll(uuid.NewString(), "-", "") + "_test"
	if _, err := control.Exec(ctx, `create database `+pgx.Identifier{name}.Sanitize()); err != nil {
		control.Close()
		t.Fatal(err)
	}
	testCfg := cfg.Copy()
	testCfg.ConnConfig.Database = name
	testCfg.MaxConns = 10
	db, err := pgxpool.NewWithConfig(ctx, testCfg)
	t.Cleanup(func() {
		if db != nil {
			db.Close()
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		if _, err := control.Exec(cleanupCtx, `drop database `+pgx.Identifier{name}.Sanitize()+` with (force)`); err != nil {
			t.Errorf("drop isolated test database: %v", err)
		}
		control.Close()
	})
	if err != nil {
		t.Fatal(err)
	}
	return db, ctx
}

type fixture struct {
	db          *pgxpool.Pool
	ctx         context.Context
	users       []auth.User
	tokens      []string
	admin       *administration.Service
	tournaments *tournament.Service
	auth        *auth.Service
	app         *fiber.App
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db, ctx := isolatedDB(t)
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	f := &fixture{db: db, ctx: ctx, admin: administration.NewService(db), tournaments: tournament.NewService(db)}
	repo := auth.NewRepository(db)
	f.auth = auth.NewService(repo, time.Hour, nil)
	for i := 0; i < 4; i++ {
		suffix := uuid.NewString()[:8]
		user, err := repo.CreateUser(ctx, auth.User{ID: uuid.New(), Email: suffix + "@example.com", Username: "u_" + suffix,
			DisplayName: "User", PasswordHash: "test-only-unused-hash", AvatarSource: auth.AvatarSourceNone,
			Role: access.Admin, RoleVersion: 999})
		if err != nil {
			t.Fatal(err)
		}
		if user.Role != access.User || user.RoleVersion != 1 {
			t.Fatalf("new user gained privilege: %+v", user)
		}
		token := uuid.NewString()
		hash := sha256.Sum256([]byte(token))
		if _, err := repo.CreateSession(ctx, auth.Session{ID: uuid.New(), UserID: user.ID,
			TokenHash: base64.RawURLEncoding.EncodeToString(hash[:]), ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
			t.Fatal(err)
		}
		f.users, f.tokens = append(f.users, user), append(f.tokens, token)
	}
	f.app = httpserver.New(config.Config{AllowedOrigins: []string{"http://frontend.test"}, MaxBodyBytes: 1 << 20,
		SubmissionRateLimit: 100, SubmissionRateWindow: time.Minute},
		auth.NewHandler(f.auth, nil, nil, "http://frontend.test", "sid", "", "Lax", "state", time.Hour, time.Minute, false),
		users.NewHandler(nil, nil, "sid"), tasks.NewHandler(nil, nil, ""), leaderboard.NewHandler(nil, nil, ""),
		matches.NewHandler(nil, time.Minute), realtime.NewHandler(realtime.NewBroker()), submissions.NewHandler(nil, "token"), nil, db,
		administration.NewHandler(f.admin), tournament.NewHandler(f.tournaments))
	return f
}

func (f *fixture) setupRoles(t *testing.T) {
	t.Helper()
	if _, err := f.admin.BootstrapAdmin(f.ctx, f.users[0].ID, "Test administrator", uuid.NewString()); err != nil {
		t.Fatal(err)
	}
	for _, user := range f.users[1:3] {
		if _, err := f.admin.ChangeRole(f.ctx, f.users[0].ID, user.ID,
			administration.ChangeRoleInput{Role: access.Organizer, ExpectedRoleVersion: 1, Reason: "Test organizer"}, uuid.NewString()); err != nil {
			t.Fatal(err)
		}
	}
}

func (f *fixture) request(t *testing.T, user int, method, path, body string, expected int) []byte {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Origin", "http://frontend.test")
	req.Header.Set("Content-Type", "application/json")
	if user >= 0 {
		req.Header.Set("Cookie", "sid="+f.tokens[user])
	}
	response, err := f.app.Test(req, 10000)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != expected {
		t.Fatalf("%s %s = %d, want %d: %s", method, path, response.StatusCode, expected, data)
	}
	if expected >= 400 && (strings.HasPrefix(path, "/api/admin/") || strings.HasPrefix(path, "/api/organizer/")) {
		var envelope struct {
			Error struct {
				Code, Message string
				RequestID     string `json:"requestId"`
			} `json:"error"`
		}
		if err := json.Unmarshal(data, &envelope); err != nil || envelope.Error.Code == "" || envelope.Error.Message == "" {
			t.Fatalf("invalid management error contract: %s", data)
		}
		if _, err := uuid.Parse(envelope.Error.RequestID); err != nil || envelope.Error.RequestID != response.Header.Get("X-Request-ID") {
			t.Fatalf("missing/mismatched management request ID: %s", data)
		}
		if expected == 401 && envelope.Error.Code != "unauthenticated" || expected == 403 && envelope.Error.Code != "forbidden" {
			t.Fatalf("incorrect auth error code: %s", data)
		}
	}
	return data
}

func (f *fixture) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var count int
	if err := f.db.QueryRow(f.ctx, query, args...).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestF1HTTPPermissionsOwnershipAndRevocation(t *testing.T) {
	f := newFixture(t)
	f.setupRoles(t)
	input := `{"slug":"autumn-cup","title":"Autumn Cup","description":"Official draft"}`
	f.request(t, -1, "POST", "/api/organizer/tournaments", input, 401)
	f.request(t, 3, "POST", "/api/organizer/tournaments", input, 403)
	f.request(t, 1, "POST", "/api/organizer/tournaments", `{"slug":"spoofed","title":"Spoofed","ownerId":"`+f.users[0].ID.String()+`"}`, 400)
	f.request(t, 1, "POST", "/api/organizer/tournaments", input+` {}`, 400)
	data := f.request(t, 1, "POST", "/api/organizer/tournaments", input, 201)
	var created struct {
		Tournament tournament.Draft `json:"tournament"`
	}
	if err := json.Unmarshal(data, &created); err != nil {
		t.Fatal(err)
	}
	draft := created.Tournament
	if draft.OwnerID != f.users[1].ID || draft.CreatedBy != f.users[1].ID || draft.ConfigVersion != 1 {
		t.Fatalf("bad ownership: %+v", draft)
	}
	path := "/api/organizer/tournaments/" + draft.ID.String()
	f.request(t, 1, "GET", path, "", 200)
	f.request(t, 2, "GET", path, "", 404)
	f.request(t, 2, "PATCH", path, `{"title":"Stolen","expectedConfigVersion":1}`, 404)
	f.request(t, 1, "PATCH", path, `{"ownerId":"`+f.users[2].ID.String()+`","expectedConfigVersion":1}`, 400)
	f.request(t, 0, "PATCH", path, `{"title":"Updated by admin","expectedConfigVersion":1}`, 200)
	f.request(t, 1, "PATCH", path, `{"title":"Stale edit","expectedConfigVersion":1}`, 409)
	f.request(t, 0, "POST", "/api/organizer/tournaments", input, 409)
	var page tournament.Page
	if err := json.Unmarshal(f.request(t, 2, "GET", "/api/organizer/tournaments", "", 200), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 0 {
		t.Fatal("organizer can list foreign drafts")
	}
	if err := json.Unmarshal(f.request(t, 0, "GET", "/api/organizer/tournaments", "", 200), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 {
		t.Fatal("admin cannot list all drafts")
	}
	if f.count(t, `select count(*) from audit_events where tournament_id=$1`, draft.ID) != 2 {
		t.Fatal("expected create/edit audit only")
	}

	rolePath := "/api/admin/users/" + f.users[1].ID.String() + "/role"
	f.request(t, 1, "PATCH", rolePath, `{"role":"user","expectedRoleVersion":2,"reason":"Self change"}`, 403)
	f.request(t, 0, "PATCH", rolePath, `{"role":"admin","expectedRoleVersion":2,"reason":"Escalation"}`, 400)
	f.request(t, 0, "PATCH", rolePath, `{"role":"user","expectedRoleVersion":2,"reason":"Organizer access revoked"}`, 200)
	f.request(t, 0, "PATCH", rolePath, `{"role":"organizer","expectedRoleVersion":2,"reason":"Stale grant"}`, 409)
	f.request(t, 1, "PATCH", path, `{"title":"After revocation","expectedConfigVersion":2}`, 403)
	f.request(t, 1, "GET", path, "", 403)
	me, err := f.auth.CurrentUser(f.ctx, f.tokens[1])
	if err != nil || me.Role != access.User || me.RoleVersion != 3 {
		t.Fatalf("stale session role: %+v, %v", me, err)
	}
	if _, err := f.tournaments.UpdateDraft(f.ctx, f.users[1].ID, draft.ID,
		tournament.UpdateInput{Title: stringPtr("Bypassed middleware"), ExpectedConfigVersion: 2}, uuid.NewString()); !errors.Is(err, access.ErrForbidden) {
		t.Fatalf("service bypass: %v", err)
	}
	stored, err := f.tournaments.GetDraft(f.ctx, f.users[0].ID, draft.ID)
	if err != nil || stored.OwnerID != f.users[1].ID || stored.Title != "Updated by admin" {
		t.Fatalf("revocation changed ownership: %+v, %v", stored, err)
	}
	if _, err := f.db.Exec(f.ctx, `update tournaments set owner_id=$2 where id=$1`, draft.ID, f.users[2].ID); err == nil {
		t.Fatal("DB allowed ownership reassignment")
	}
}

func TestF1SignupCannotChooseRole(t *testing.T) {
	f := newFixture(t)
	data := f.request(t, -1, "POST", "/auth/sign-up", `{"username":"signup_player","email":"signup@example.com","password":"StrongPassword123!","role":"admin","roleVersion":999}`, 201)
	var response struct {
		User auth.PublicUser `json:"user"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	if response.User.Role != access.User || response.User.RoleVersion != 1 {
		t.Fatalf("signup escalation: %+v", response.User)
	}
}

func TestF1DraftListExcludesOtherStatesWithPagination(t *testing.T) {
	f := newFixture(t)
	f.setupRoles(t)
	if _, err := f.db.Exec(f.ctx, `alter table tournaments drop constraint tournaments_status_check;
        alter table tournaments add constraint tournaments_status_check
        check (status in ('draft', 'announced', 'running', 'finished'))`); err != nil {
		t.Fatal(err)
	}
	create := func(owner int, slug string) tournament.Draft {
		t.Helper()
		draft, err := f.tournaments.CreateDraft(f.ctx, f.users[owner].ID,
			tournament.CreateInput{Slug: slug, Title: slug}, uuid.NewString())
		if err != nil {
			t.Fatal(err)
		}
		return draft
	}
	first := create(1, "first-draft")
	otherOwner := create(2, "other-owner-draft")
	last := create(1, "last-draft")
	for _, status := range []string{"announced", "running", "finished"} {
		draft := create(1, "non-draft-"+status)
		if _, err := f.db.Exec(f.ctx, `update tournaments set status=$2 where id=$1`, draft.ID, status); err != nil {
			t.Fatal(err)
		}
	}
	for _, test := range []struct {
		name  string
		actor int
		ids   []uuid.UUID
	}{
		{"admin", 0, []uuid.UUID{last.ID, otherOwner.ID, first.ID}},
		{"owner", 1, []uuid.UUID{last.ID, first.ID}},
		{"other organizer", 2, []uuid.UUID{otherOwner.ID}},
	} {
		t.Run(test.name, func(t *testing.T) {
			cursor := ""
			for i, expectedID := range test.ids {
				var page tournament.Page
				body := f.request(t, test.actor, "GET", "/api/organizer/tournaments?limit=1&cursor="+cursor, "", 200)
				if err := json.Unmarshal(body, &page); err != nil {
					t.Fatal(err)
				}
				if len(page.Items) != 1 || page.Items[0].ID != expectedID || page.Items[0].Status != "draft" {
					t.Fatalf("unexpected draft page: %s", body)
				}
				if (page.NextCursor != "") != (i < len(test.ids)-1) {
					t.Fatalf("unexpected pagination cursor: %s", body)
				}
				cursor = page.NextCursor
			}
		})
	}
}

func TestF1AuditFailureRollsBackAndAuditIsImmutable(t *testing.T) {
	f := newFixture(t)
	f.setupRoles(t)
	draft, err := f.tournaments.CreateDraft(f.ctx, f.users[1].ID, tournament.CreateInput{Slug: "audit-cup", Title: "Audit Cup"}, uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{`update audit_events set metadata='{}'`, `delete from audit_events`, `truncate audit_events`} {
		if _, err := f.db.Exec(f.ctx, query); err == nil {
			t.Fatalf("audit mutation succeeded: %s", query)
		}
	}
	before := f.count(t, `select count(*) from audit_events`)
	_, err = f.db.Exec(f.ctx, `create function fail_audit_test() returns trigger language plpgsql as $$ begin raise exception 'injected audit failure'; end $$;
        create trigger fail_audit_test before insert on audit_events for each row execute function fail_audit_test()`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.admin.ChangeRole(f.ctx, f.users[0].ID, f.users[1].ID, administration.ChangeRoleInput{
		Role: access.User, ExpectedRoleVersion: 2, Reason: "Must roll back"}, uuid.NewString()); err == nil {
		t.Fatal("role change ignored audit failure")
	}
	actor, err := f.auth.CurrentUser(f.ctx, f.tokens[1])
	if err != nil || actor.Role != access.Organizer || actor.RoleVersion != 2 {
		t.Fatalf("role mutation escaped rollback: %+v %v", actor, err)
	}
	if _, err := f.tournaments.UpdateDraft(f.ctx, f.users[1].ID, draft.ID, tournament.UpdateInput{
		Title: stringPtr("Must not persist"), ExpectedConfigVersion: 1}, uuid.NewString()); err == nil {
		t.Fatal("edit ignored audit failure")
	}
	if _, err := f.tournaments.CreateDraft(f.ctx, f.users[1].ID, tournament.CreateInput{Slug: "failed-cup", Title: "Failed"}, uuid.NewString()); err == nil {
		t.Fatal("creation ignored audit failure")
	}
	stored, err := f.tournaments.GetDraft(f.ctx, f.users[1].ID, draft.ID)
	if err != nil || stored.Title != draft.Title || stored.ConfigVersion != 1 {
		t.Fatalf("edit escaped rollback: %+v %v", stored, err)
	}
	if f.count(t, `select count(*) from tournaments`) != 1 || f.count(t, `select count(*) from audit_events`) != before {
		t.Fatal("partial mutation/audit commit")
	}
}

func TestF1ConcurrentBootstrapAndRoleCAS(t *testing.T) {
	f := newFixture(t)
	start := make(chan struct{})
	result := make(chan error, 2)
	for _, user := range f.users[:2] {
		go func(id uuid.UUID) {
			<-start
			_, err := f.admin.BootstrapAdmin(f.ctx, id, "Concurrent bootstrap", uuid.NewString())
			result <- err
		}(user.ID)
	}
	close(start)
	success, rejected := 0, 0
	for i := 0; i < 2; i++ {
		err := <-result
		if err == nil {
			success++
		} else if errors.Is(err, administration.ErrAdminAlreadyExists) {
			rejected++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || rejected != 1 || f.count(t, `select count(*) from users where role='admin'`) != 1 {
		t.Fatal("bootstrap created multiple admins")
	}
	var adminID uuid.UUID
	if err := f.db.QueryRow(f.ctx, `select id from users where role='admin'`).Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	start = make(chan struct{})
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			_, err := f.admin.ChangeRole(f.ctx, adminID, f.users[2].ID, administration.ChangeRoleInput{
				Role: access.Organizer, ExpectedRoleVersion: 1, Reason: "Concurrent grant"}, uuid.NewString())
			result <- err
		}()
	}
	close(start)
	success, rejected = 0, 0
	for i := 0; i < 2; i++ {
		err := <-result
		if err == nil {
			success++
		} else if errors.Is(err, access.ErrConflict) {
			rejected++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || rejected != 1 || f.count(t, `select count(*) from audit_events where entity_id=$1 and action='role.granted'`, f.users[2].ID) != 1 {
		t.Fatal("concurrent role writes were not serialized")
	}
	if _, err := f.admin.ChangeRole(f.ctx, adminID, adminID, administration.ChangeRoleInput{
		Role: access.User, ExpectedRoleVersion: 2, Reason: "Demote admin"}, uuid.NewString()); !errors.Is(err, access.ErrForbidden) {
		t.Fatalf("admin demotion allowed: %v", err)
	}
}

func TestF1RevocationAndProtectedWriteAreOrdered(t *testing.T) {
	for _, revokeFirst := range []bool{false, true} {
		name := "write_before_revoke"
		if revokeFirst {
			name = "revoke_before_write"
		}
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			f.setupRoles(t)
			gate, err := f.db.Begin(f.ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer gate.Rollback(f.ctx)
			if _, err := gate.Exec(f.ctx, `select pg_advisory_xact_lock(172937,1)`); err != nil {
				t.Fatal(err)
			}
			trigger := `create function gate_test_write() returns trigger language plpgsql as $$ begin perform pg_advisory_xact_lock(172937,1); return new; end $$;
                create trigger gate_test_write before insert on tournaments for each row execute function gate_test_write()`
			if revokeFirst {
				trigger = `create function gate_test_write() returns trigger language plpgsql as $$ begin if new.action='role.revoked' then perform pg_advisory_xact_lock(172937,1); end if; return new; end $$;
                    create trigger gate_test_write before insert on audit_events for each row execute function gate_test_write()`
			}
			if _, err := f.db.Exec(f.ctx, trigger); err != nil {
				t.Fatal(err)
			}
			writeResult, revokeResult := make(chan error, 1), make(chan error, 1)
			write := func() {
				_, err := f.tournaments.CreateDraft(f.ctx, f.users[1].ID, tournament.CreateInput{Slug: "race-cup", Title: "Race Cup"}, uuid.NewString())
				writeResult <- err
			}
			revoke := func() {
				_, err := f.admin.ChangeRole(f.ctx, f.users[0].ID, f.users[1].ID, administration.ChangeRoleInput{
					Role: access.User, ExpectedRoleVersion: 2, Reason: "Race revocation"}, uuid.NewString())
				revokeResult <- err
			}
			if revokeFirst {
				go revoke()
			} else {
				go write()
			}
			waitForBlocked(t, f, 1)
			if revokeFirst {
				go write()
			} else {
				go revoke()
			}
			waitForBlocked(t, f, 2)
			if err := gate.Commit(f.ctx); err != nil {
				t.Fatal(err)
			}
			if err := <-revokeResult; err != nil {
				t.Fatal(err)
			}
			writeErr := <-writeResult
			if revokeFirst && !errors.Is(writeErr, access.ErrForbidden) {
				t.Fatalf("write after revocation: %v", writeErr)
			}
			if !revokeFirst && writeErr != nil {
				t.Fatalf("admitted write failed: %v", writeErr)
			}
			expected := 1
			if revokeFirst {
				expected = 0
			}
			if f.count(t, `select count(*) from tournaments`) != expected {
				t.Fatal("incorrect race outcome")
			}
		})
	}
}

func waitForBlocked(t *testing.T, f *fixture, expected int) {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		count := f.count(t, `select count(*) from pg_stat_activity where datname=current_database() and wait_event_type='Lock' and cardinality(pg_blocking_pids(pid))>0`)
		if count >= expected {
			return
		}
		select {
		case <-deadline.C:
			t.Fatalf("expected %d blocked transactions, observed %d", expected, count)
		case <-ticker.C:
		}
	}
}

func TestF1UpgradePreservesExistingUsers(t *testing.T) {
	db, ctx := isolatedDB(t)
	if _, err := db.Exec(ctx, `create table schema_migrations(name text primary key, applied_at timestamptz not null default now())`); err != nil {
		t.Fatal(err)
	}
	files, err := os.ReadDir(filepath.Join("..", "database", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if file.IsDir() || file.Name() >= "029" {
			continue
		}
		sql, err := os.ReadFile(filepath.Join("..", "database", "migrations", file.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("legacy migration %s: %v", file.Name(), err)
		}
		if _, err := db.Exec(ctx, `insert into schema_migrations(name) values($1)`, file.Name()); err != nil {
			t.Fatal(err)
		}
	}
	id := uuid.New()
	if _, err := db.Exec(ctx, `insert into users(id,email,username,password_hash) values($1,'legacy@example.com','legacy','hash')`, id); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	user, err := auth.NewRepository(db).GetUserByID(ctx, id)
	if err != nil || user.Role != access.User || user.RoleVersion != 1 || user.Username != "legacy" {
		t.Fatalf("bad upgrade: %+v %v", user, err)
	}
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
}

func stringPtr(value string) *string { return &value }
