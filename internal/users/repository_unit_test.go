package users

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestNewRepository(t *testing.T) {
	if repo := NewRepository(nil); repo == nil {
		t.Fatal("NewRepository returned nil")
	}
}

func TestRepositoryBuildsProfileFromRows(t *testing.T) {
	userID := uuid.New()
	opponentID := uuid.New()
	matchID := uuid.New()
	now := time.Now().UTC()
	globalRank := 3
	globalPlayers := 100
	regionalRank := 1
	regionalPlayers := 10
	eloDelta := 24
	difficulty := "hard"

	db := &fakeUsersDB{
		rows: []pgx.Row{
			fakeUsersRow{values: []any{
				userID,
				"ada",
				"",
				"bio",
				"Germany",
				"DE",
				"Berlin",
				"Berlin",
				"https://custom",
				"avatars/ada.png",
				"https://provider",
				AvatarSourceCustom,
				"github",
				now,
			}},
			fakeUsersRow{values: []any{1800, 4, 3, 1, 0}},
			fakeUsersRow{values: []any{globalRank, globalPlayers}},
			fakeUsersRow{values: []any{regionalRank, regionalPlayers}},
		},
		queryRows: []pgx.Rows{
			&fakeUsersRows{rows: [][]any{{&userID}, {&opponentID}, {&userID}}},
			&fakeUsersRows{rows: [][]any{{"2026-01-01", 20, 1820, 2}}},
			&fakeUsersRows{rows: [][]any{{"typescript", 0.6}, {"go", 0.4}}},
			&fakeUsersRows{rows: [][]any{{
				matchID,
				"win",
				opponentID,
				"grace",
				"",
				"https://opponent",
				&difficulty,
				&eloDelta,
				true,
				now,
			}}},
		},
	}
	repo := &Repository{db: db}

	profile, err := repo.GetProfileByUsername(context.Background(), "ada")
	if err != nil {
		t.Fatalf("GetProfileByUsername returned error: %v", err)
	}
	if profile.ID != userID || profile.AvatarURL != "https://custom" || profile.DisplayName != "" {
		t.Fatalf("profile basics = %#v", profile)
	}
	if profile.Stats.EloRating != 1800 || profile.Stats.WinRate != 0.75 ||
		profile.Stats.GlobalRank == nil || *profile.Stats.GlobalRank != 3 ||
		profile.Stats.RegionalRank == nil || *profile.Stats.RegionalRank != 1 ||
		profile.Stats.MaxWinStreak != 1 {
		t.Fatalf("stats = %#v", profile.Stats)
	}
	if len(profile.EloHistory) != 1 || profile.EloHistory[0].Rating != 1820 {
		t.Fatalf("elo history = %#v", profile.EloHistory)
	}
	if len(profile.TopLanguages) != 2 || profile.TopLanguages[0].Name != "TypeScript" || profile.TopLanguages[1].Name != "Go" {
		t.Fatalf("top languages = %#v", profile.TopLanguages)
	}
	if len(profile.RecentMatches) != 1 || profile.RecentMatches[0].Opponent.DisplayName != "grace" {
		t.Fatalf("recent matches = %#v", profile.RecentMatches)
	}
}

func TestRepositoryUpdateProfileAndAvatarHelpers(t *testing.T) {
	userID := uuid.New()
	db := &fakeUsersDB{
		rows: []pgx.Row{
			fakeUsersRow{values: []any{"old/key.png"}},
		},
	}
	repo := &Repository{db: db}

	key, err := repo.GetAvatarObjectKey(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetAvatarObjectKey returned error: %v", err)
	}
	if key != "old/key.png" {
		t.Fatalf("avatar key = %q", key)
	}

	if got := resolveAvatarURL(AvatarSourceProvider, "custom", "provider"); got != "provider" {
		t.Fatalf("provider avatar = %q", got)
	}
	if got := resolveAvatarURL(AvatarSourceNone, "", "provider"); got != "provider" {
		t.Fatalf("fallback avatar = %q", got)
	}
	if got := displayLanguageName(" csharp "); got != "C#" {
		t.Fatalf("display language = %q", got)
	}
}

func TestDisplayLanguageName(t *testing.T) {
	tests := map[string]string{
		"typescript": "TypeScript",
		"javascript": "JavaScript",
		"python":     "Python",
		"java":       "Java",
		"cpp":        "C++",
		"go":         "Go",
		"rust":       "Rust",
		"zig":        "Zig",
		"csharp":     "C#",
		"c":          "C",
		"ruby":       "Ruby",
		"php":        "PHP",
		"sql":        "SQL",
		"elixir":     "elixir",
	}
	for input, want := range tests {
		if got := displayLanguageName(input); got != want {
			t.Fatalf("displayLanguageName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRepositoryUpdateProfile(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()
	db := &fakeUsersDB{
		rows: []pgx.Row{
			fakeUsersRow{values: []any{userID}},
			fakeUsersRow{values: fakeProfileRecordValues(userID, "ada", "Ada", AvatarSourceNone, "", "", now)},
			fakeUsersRow{values: []any{1000, 0, 0, 0, 0}},
			fakeUsersRow{err: pgx.ErrNoRows},
		},
		queryRows: []pgx.Rows{
			&fakeUsersRows{},
			&fakeUsersRows{},
			&fakeUsersRows{},
		},
	}
	repo := &Repository{db: db}

	profile, err := repo.UpdateProfile(context.Background(), userID, "ada", UpdateProfileInput{
		DisplayName: "Ada",
		Bio:         "solver",
		Country:     "Germany",
		CountryCode: "DE",
	})
	if err != nil {
		t.Fatalf("UpdateProfile returned error: %v", err)
	}
	if profile.ID != userID || profile.DisplayName != "Ada" || profile.Stats.EloRating != 1000 {
		t.Fatalf("profile = %#v", profile)
	}
}

func TestRepositoryUpdateProfileRejectsNonOwner(t *testing.T) {
	repo := &Repository{db: &fakeUsersDB{rows: []pgx.Row{fakeUsersRow{values: []any{uuid.New()}}}}}

	_, err := repo.UpdateProfile(context.Background(), uuid.New(), "ada", UpdateProfileInput{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("UpdateProfile error = %v, want ErrForbidden", err)
	}
}

func TestRepositoryUpdateCustomAvatar(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()
	db := &fakeUsersDB{
		rows: []pgx.Row{
			fakeUsersRow{values: []any{userID}},
			fakeUsersRow{values: []any{"avatars/old.png"}},
			fakeUsersRow{values: fakeProfileRecordValues(userID, "ada", "Ada", AvatarSourceCustom, "https://cdn/new.png", "https://provider", now)},
			fakeUsersRow{values: []any{1200, 2, 1, 1, 0}},
			fakeUsersRow{values: []any{2, 20}},
		},
		queryRows: []pgx.Rows{
			&fakeUsersRows{rows: [][]any{{&userID}}},
			&fakeUsersRows{},
			&fakeUsersRows{},
			&fakeUsersRows{},
		},
	}
	repo := &Repository{db: db}

	profile, oldKey, err := repo.UpdateCustomAvatar(context.Background(), userID, "ada", "https://cdn/new.png", "avatars/new.png")
	if err != nil {
		t.Fatalf("UpdateCustomAvatar returned error: %v", err)
	}
	if oldKey != "avatars/old.png" || profile.AvatarURL != "https://cdn/new.png" || profile.Stats.MaxWinStreak != 1 {
		t.Fatalf("profile=%#v oldKey=%q", profile, oldKey)
	}
}

func TestRepositoryAvatarNotFoundBranches(t *testing.T) {
	repo := &Repository{db: &fakeUsersDB{rows: []pgx.Row{fakeUsersRow{err: pgx.ErrNoRows}}}}
	if _, err := repo.GetAvatarObjectKey(context.Background(), uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetAvatarObjectKey error = %v", err)
	}

	userID := uuid.New()
	repo = &Repository{db: &fakeUsersDB{rows: []pgx.Row{
		fakeUsersRow{values: []any{userID}},
		fakeUsersRow{err: pgx.ErrNoRows},
	}}}
	if _, _, err := repo.UpdateCustomAvatar(context.Background(), userID, "ada", "url", "key"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateCustomAvatar error = %v", err)
	}
}

func TestScanProfileRecordMapsNotFound(t *testing.T) {
	if _, err := scanProfileRecord(fakeUsersRow{err: pgx.ErrNoRows}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("scanProfileRecord error = %v, want ErrNotFound", err)
	}
}

func TestRepositoryHelperErrorBranches(t *testing.T) {
	userID := uuid.New()
	dbErr := errors.New("db failed")

	tests := []struct {
		name string
		repo *Repository
		call func(*Repository) error
		want string
	}{
		{
			name: "user id generic",
			repo: &Repository{db: &fakeUsersDB{rows: []pgx.Row{fakeUsersRow{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, err := repo.getUserIDByUsername(context.Background(), "ada")
				return err
			},
			want: "get user id by username",
		},
		{
			name: "avatar generic",
			repo: &Repository{db: &fakeUsersDB{rows: []pgx.Row{fakeUsersRow{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, err := repo.GetAvatarObjectKey(context.Background(), userID)
				return err
			},
			want: "get avatar object key",
		},
		{
			name: "update profile exec",
			repo: &Repository{db: &fakeUsersDB{rows: []pgx.Row{fakeUsersRow{values: []any{userID}}}, execErr: dbErr}},
			call: func(repo *Repository) error {
				_, err := repo.UpdateProfile(context.Background(), userID, "ada", UpdateProfileInput{})
				return err
			},
			want: "update profile",
		},
		{
			name: "custom avatar generic",
			repo: &Repository{db: &fakeUsersDB{rows: []pgx.Row{fakeUsersRow{values: []any{userID}}, fakeUsersRow{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, _, err := repo.UpdateCustomAvatar(context.Background(), userID, "ada", "url", "key")
				return err
			},
			want: "update custom avatar",
		},
		{
			name: "stats generic",
			repo: &Repository{db: &fakeUsersDB{rows: []pgx.Row{fakeUsersRow{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, err := repo.getProfileStats(context.Background(), userID, "", "")
				return err
			},
			want: "get profile stats",
		},
		{
			name: "global rank generic",
			repo: &Repository{db: &fakeUsersDB{rows: []pgx.Row{fakeUsersRow{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, _, err := repo.getGlobalRank(context.Background(), userID)
				return err
			},
			want: "get global rank",
		},
		{
			name: "regional rank generic",
			repo: &Repository{db: &fakeUsersDB{rows: []pgx.Row{fakeUsersRow{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, _, err := repo.getRegionalRank(context.Background(), userID, "DE", "Germany")
				return err
			},
			want: "get regional rank",
		},
		{
			name: "streak query",
			repo: &Repository{db: &fakeUsersDB{queryErr: dbErr}},
			call: func(repo *Repository) error {
				_, err := repo.getMaxWinStreak(context.Background(), userID)
				return err
			},
			want: "list matches for streak",
		},
		{
			name: "streak scan",
			repo: &Repository{db: &fakeUsersDB{queryRows: []pgx.Rows{&fakeUsersRows{rows: [][]any{{"bad"}}}}}},
			call: func(repo *Repository) error {
				_, err := repo.getMaxWinStreak(context.Background(), userID)
				return err
			},
			want: "scan streak match",
		},
		{
			name: "streak iterate",
			repo: &Repository{db: &fakeUsersDB{queryRows: []pgx.Rows{&fakeUsersRows{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, err := repo.getMaxWinStreak(context.Background(), userID)
				return err
			},
			want: "iterate streak matches",
		},
		{
			name: "elo query",
			repo: &Repository{db: &fakeUsersDB{queryErr: dbErr}},
			call: func(repo *Repository) error {
				_, err := repo.getEloHistory(context.Background(), userID)
				return err
			},
			want: "get elo history",
		},
		{
			name: "elo scan",
			repo: &Repository{db: &fakeUsersDB{queryRows: []pgx.Rows{&fakeUsersRows{rows: [][]any{{"date"}}}}}},
			call: func(repo *Repository) error {
				_, err := repo.getEloHistory(context.Background(), userID)
				return err
			},
			want: "scan elo history",
		},
		{
			name: "elo iterate",
			repo: &Repository{db: &fakeUsersDB{queryRows: []pgx.Rows{&fakeUsersRows{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, err := repo.getEloHistory(context.Background(), userID)
				return err
			},
			want: "iterate elo history",
		},
		{
			name: "languages query",
			repo: &Repository{db: &fakeUsersDB{queryErr: dbErr}},
			call: func(repo *Repository) error {
				_, err := repo.getTopLanguages(context.Background(), userID)
				return err
			},
			want: "get top languages",
		},
		{
			name: "languages scan",
			repo: &Repository{db: &fakeUsersDB{queryRows: []pgx.Rows{&fakeUsersRows{rows: [][]any{{"go"}}}}}},
			call: func(repo *Repository) error {
				_, err := repo.getTopLanguages(context.Background(), userID)
				return err
			},
			want: "scan top language",
		},
		{
			name: "languages iterate",
			repo: &Repository{db: &fakeUsersDB{queryRows: []pgx.Rows{&fakeUsersRows{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, err := repo.getTopLanguages(context.Background(), userID)
				return err
			},
			want: "iterate top languages",
		},
		{
			name: "recent query",
			repo: &Repository{db: &fakeUsersDB{queryErr: dbErr}},
			call: func(repo *Repository) error {
				_, err := repo.getRecentMatches(context.Background(), userID)
				return err
			},
			want: "get recent matches",
		},
		{
			name: "recent scan",
			repo: &Repository{db: &fakeUsersDB{queryRows: []pgx.Rows{&fakeUsersRows{rows: [][]any{{uuid.New()}}}}}},
			call: func(repo *Repository) error {
				_, err := repo.getRecentMatches(context.Background(), userID)
				return err
			},
			want: "scan recent match",
		},
		{
			name: "recent iterate",
			repo: &Repository{db: &fakeUsersDB{queryRows: []pgx.Rows{&fakeUsersRows{err: dbErr}}}},
			call: func(repo *Repository) error {
				_, err := repo.getRecentMatches(context.Background(), userID)
				return err
			},
			want: "iterate recent matches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call(tt.repo)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestRepositoryNoRowsBranches(t *testing.T) {
	repo := &Repository{db: &fakeUsersDB{rows: []pgx.Row{fakeUsersRow{err: pgx.ErrNoRows}}}}
	if _, err := repo.getUserIDByUsername(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("getUserIDByUsername error = %v", err)
	}

	repo = &Repository{db: &fakeUsersDB{rows: []pgx.Row{fakeUsersRow{err: pgx.ErrNoRows}}}}
	stats, err := repo.getProfileStats(context.Background(), uuid.New(), "", "")
	if err != nil || stats.EloRating != 1000 {
		t.Fatalf("no rows stats = %#v, %v", stats, err)
	}

	repo = &Repository{db: &fakeUsersDB{rows: []pgx.Row{fakeUsersRow{err: pgx.ErrNoRows}}}}
	rank, players, err := repo.getGlobalRank(context.Background(), uuid.New())
	if err != nil || rank != nil || players != nil {
		t.Fatalf("global no rows = %v/%v/%v", rank, players, err)
	}

	repo = &Repository{db: &fakeUsersDB{rows: []pgx.Row{fakeUsersRow{err: pgx.ErrNoRows}}}}
	rank, players, err = repo.getRegionalRank(context.Background(), uuid.New(), "DE", "Germany")
	if err != nil || rank != nil || players != nil {
		t.Fatalf("regional no rows = %v/%v/%v", rank, players, err)
	}
}

func fakeProfileRecordValues(
	userID uuid.UUID,
	username string,
	displayName string,
	avatarSource AvatarSource,
	avatarURL string,
	providerAvatarURL string,
	createdAt time.Time,
) []any {
	return []any{
		userID,
		username,
		displayName,
		"bio",
		"Germany",
		"DE",
		"Berlin",
		"Berlin",
		avatarURL,
		"avatars/current.png",
		providerAvatarURL,
		avatarSource,
		"github",
		createdAt,
	}
}

type fakeUsersDB struct {
	rows      []pgx.Row
	queryRows []pgx.Rows
	execErr   error
	queryErr  error
}

func (db *fakeUsersDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	if db.execErr != nil {
		return pgconn.CommandTag{}, db.execErr
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (db *fakeUsersDB) QueryRow(context.Context, string, ...any) pgx.Row {
	if len(db.rows) == 0 {
		return fakeUsersRow{err: pgx.ErrNoRows}
	}
	row := db.rows[0]
	db.rows = db.rows[1:]
	return row
}

func (db *fakeUsersDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	if db.queryErr != nil {
		return nil, db.queryErr
	}
	if len(db.queryRows) == 0 {
		return &fakeUsersRows{}, nil
	}
	rows := db.queryRows[0]
	db.queryRows = db.queryRows[1:]
	return rows, nil
}

type fakeUsersRows struct {
	rows [][]any
	idx  int
	err  error
}

func (r *fakeUsersRows) Close()                                       {}
func (r *fakeUsersRows) Err() error                                   { return r.err }
func (r *fakeUsersRows) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("SELECT 1") }
func (r *fakeUsersRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeUsersRows) Next() bool {
	if r.idx >= len(r.rows) {
		return false
	}
	r.idx++
	return true
}
func (r *fakeUsersRows) Scan(dest ...any) error { return scanFakeUsersValues(r.rows[r.idx-1], dest...) }
func (r *fakeUsersRows) Values() ([]any, error) { return r.rows[r.idx-1], nil }
func (r *fakeUsersRows) RawValues() [][]byte    { return nil }
func (r *fakeUsersRows) Conn() *pgx.Conn        { return nil }

type fakeUsersRow struct {
	values []any
	err    error
}

func (r fakeUsersRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	return scanFakeUsersValues(r.values, dest...)
}

func scanFakeUsersValues(values []any, dest ...any) error {
	if len(values) != len(dest) {
		return errors.New("fake row destination count mismatch")
	}
	for i := range dest {
		target := reflect.ValueOf(dest[i])
		if target.Kind() != reflect.Pointer || target.IsNil() {
			return errors.New("fake row destination must be pointer")
		}
		value := reflect.ValueOf(values[i])
		elem := target.Elem()
		if !value.IsValid() {
			elem.Set(reflect.Zero(elem.Type()))
			continue
		}
		if value.Type().AssignableTo(elem.Type()) {
			elem.Set(value)
			continue
		}
		if value.Type().ConvertibleTo(elem.Type()) {
			elem.Set(value.Convert(elem.Type()))
			continue
		}
		return errors.New("fake row value type mismatch")
	}
	return nil
}
