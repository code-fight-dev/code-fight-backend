package notifications

import (
	"testing"

	"code-fight-backend/internal/testsupport"

	"github.com/google/uuid"
)

func TestScheduledNotificationsAndNoShow(t *testing.T) {
	db, ctx := testsupport.RequirePostgres(t)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	ready, absent, organizer := uuid.New(), uuid.New(), uuid.New()
	if ready.String() > absent.String() {
		ready, absent = absent, ready
	}
	for _, user := range []uuid.UUID{ready, absent, organizer} {
		exec("insert into users(id,email,username,password_hash) values($1,$2,$3,'fixture')",
			user, user.String()+"@example.com", "u_"+user.String()[:12])
	}
	tournamentID, matchID, seen, unseen := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, "delete from tournaments where id=$1", tournamentID)
		_, _ = db.Exec(ctx, "delete from matches where id=$1", matchID)
		_, _ = db.Exec(ctx, "delete from submissions where id in ($1,$2)", seen, unseen)
		_, _ = db.Exec(ctx, "delete from users where id in ($1,$2,$3)", ready, absent, organizer)
	})
	exec(`insert into tournaments(id,slug,title,owner_id,created_by,status,task_id)
	    select $1::uuid,$1::uuid::text,'Local cup',$2,$2,'running',id from tasks where slug='signal-stabilizer'`, tournamentID, organizer)
	exec("insert into tournament_participants values($1,$2),($1,$3)", tournamentID, ready, absent)
	exec(`insert into matches(id,player1_id,player2_id,player1_ready,is_rated,created_at)
	    values($1,$2,$3,true,false,now()-interval '13 minutes')`, matchID, ready, absent)
	exec(`insert into tournament_fixtures(tournament_id,player1_id,player2_id,match_id)
	    values($1,$2,$3,$4)`, tournamentID, ready, absent, matchID)

	for _, id := range []uuid.UUID{seen, unseen} {
		exec(`insert into submissions(id,user_id,task_id,language,language_version,source_code)
		    select $1,$2,id,'python','3.12','return 42' from tasks where slug='signal-stabilizer'`, id, ready)
		exec("update submissions set status='finished',verdict='accepted' where id=$1", id)
	}
	store := NewStore(db)
	if err := store.DismissPendingSubmission(ctx, absent, unseen); err != nil {
		t.Fatal(err)
	}
	if err := store.DismissPendingSubmission(ctx, ready, seen); err != nil {
		t.Fatal(err)
	}
	exec("update notifications set deliver_at=now()-interval '1 second' where event_key=$1", unseen.String())
	for range 2 {
		if err := deliverScheduled(ctx, db, "15 minutes"); err != nil {
			t.Fatal(err)
		}
	}

	exec("update matches set status='cancelled' where id=$1", matchID)
	exec("update users set role='organizer',role_version=role_version+1 where id=$1", organizer)
	expected := map[uuid.UUID][]string{
		ready:     {"match.cancelled", "submission.reviewed"},
		absent:    {"tournament.ready_reminder"},
		organizer: {"account.role_changed", "tournament.match_cancelled"},
	}
	for user, kinds := range expected {
		page, err := store.List(ctx, user, nil)
		if err != nil || len(page.Items) != len(kinds) || page.Unread != len(kinds) {
			t.Fatalf("inbox for %s: %#v, %v", user, page, err)
		}
		for i, kind := range kinds {
			if page.Items[i].Kind != kind {
				t.Fatalf("got %s, want %s", page.Items[i].Kind, kind)
			}
		}
	}
}
