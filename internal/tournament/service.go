package tournament

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"code-fight-backend/internal/access"
	"code-fight-backend/internal/audit"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type database interface {
	Begin(context.Context) (pgx.Tx, error)
}
type Service struct{ db database }

func NewService(db database) *Service { return &Service{db: db} }

type Draft struct {
	ID            uuid.UUID `json:"id"`
	Slug          string    `json:"slug"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	OwnerID       uuid.UUID `json:"ownerId"`
	CreatedBy     uuid.UUID `json:"createdBy"`
	Status        string    `json:"status"`
	ConfigVersion int64     `json:"configVersion"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type CreateInput struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
}
type UpdateInput struct {
	Slug                  *string `json:"slug"`
	Title                 *string `json:"title"`
	Description           *string `json:"description"`
	ExpectedConfigVersion int64   `json:"expectedConfigVersion"`
}
type Page struct {
	Items      []Draft `json:"items"`
	NextCursor string  `json:"nextCursor,omitempty"`
}
type cursor struct {
	CreatedAt time.Time `json:"createdAt"`
	ID        uuid.UUID `json:"id"`
}

var ErrSlugTaken = errors.New("tournament slug already exists")
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

const columns = `id, slug, title, description, owner_id, created_by, status, config_version, created_at, updated_at`

func scanDraft(row interface{ Scan(...any) error }) (Draft, error) {
	var draft Draft
	err := row.Scan(&draft.ID, &draft.Slug, &draft.Title, &draft.Description, &draft.OwnerID,
		&draft.CreatedBy, &draft.Status, &draft.ConfigVersion, &draft.CreatedAt, &draft.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Draft{}, access.ErrNotFound
	}
	return draft, mapError(err)
}

func mapError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "tournaments_slug_key" {
		return ErrSlugTaken
	}
	return err
}

func normalize(input CreateInput) (CreateInput, error) {
	input.Slug = strings.ToLower(strings.TrimSpace(input.Slug))
	input.Title = strings.TrimSpace(input.Title)
	if len(input.Slug) < 3 || len(input.Slug) > 100 || !slugPattern.MatchString(input.Slug) ||
		utf8.RuneCountInString(input.Title) < 1 || utf8.RuneCountInString(input.Title) > 200 ||
		utf8.RuneCountInString(input.Description) > 20000 {
		return CreateInput{}, access.ErrInvalidInput
	}
	return input, nil
}

func (s *Service) CreateDraft(ctx context.Context, actorID uuid.UUID, input CreateInput, requestID string) (Draft, error) {
	input, err := normalize(input)
	if err != nil {
		return Draft{}, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Draft{}, err
	}
	defer tx.Rollback(ctx)
	actor, err := access.LockActor(ctx, tx, actorID)
	if err != nil {
		return Draft{}, err
	}
	if !access.CanCreateTournament(actor.Role) {
		return Draft{}, access.ErrForbidden
	}
	draft, err := scanDraft(tx.QueryRow(ctx, `insert into tournaments (id,slug,title,description,owner_id,created_by)
        values ($1,$2,$3,$4,$5,$5) returning `+columns, uuid.New(), input.Slug, input.Title, input.Description, actorID))
	if err != nil {
		return Draft{}, err
	}
	if err := audit.Append(ctx, tx, audit.Event{
		ActorID: &actorID, Action: "tournament.created", EntityType: "tournament", EntityID: draft.ID,
		TournamentID: &draft.ID, RequestID: requestID,
		Metadata: map[string]any{"slug": draft.Slug, "status": draft.Status, "configVersion": draft.ConfigVersion},
	}); err != nil {
		return Draft{}, err
	}
	return draft, tx.Commit(ctx)
}

func (s *Service) GetDraft(ctx context.Context, actorID, id uuid.UUID) (Draft, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Draft{}, err
	}
	defer tx.Rollback(ctx)
	actor, err := access.LockActor(ctx, tx, actorID)
	if err != nil {
		return Draft{}, err
	}
	if !access.CanCreateTournament(actor.Role) {
		return Draft{}, access.ErrForbidden
	}
	draft, err := scanDraft(tx.QueryRow(ctx, `select `+columns+` from tournaments where id=$1`, id))
	if err != nil {
		return Draft{}, err
	}
	if !access.CanManageTournament(actor.Role, actor.ID, draft.OwnerID) {
		return Draft{}, access.ErrNotFound
	}
	return draft, tx.Commit(ctx)
}

func (s *Service) ListDrafts(ctx context.Context, actorID uuid.UUID, limit int, rawCursor string) (Page, error) {
	if limit < 1 || limit > 100 || len(rawCursor) > 256 {
		return Page{}, access.ErrInvalidInput
	}
	var before *time.Time
	var after cursor
	if rawCursor != "" {
		data, err := base64.RawURLEncoding.DecodeString(rawCursor)
		if err != nil || json.Unmarshal(data, &after) != nil || after.ID == uuid.Nil || after.CreatedAt.IsZero() {
			return Page{}, access.ErrInvalidInput
		}
		before = &after.CreatedAt
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Page{}, err
	}
	defer tx.Rollback(ctx)
	actor, err := access.LockActor(ctx, tx, actorID)
	if err != nil {
		return Page{}, err
	}
	if !access.CanCreateTournament(actor.Role) {
		return Page{}, access.ErrForbidden
	}
	rows, err := tx.Query(ctx, `select `+columns+` from tournaments
        where status='draft' and ($1::boolean or owner_id=$2)
          and ($3::timestamptz is null or (created_at,id)<($3,$4))
        order by created_at desc, id desc limit $5`, actor.Role == access.Admin, actor.ID, before, after.ID, limit+1)
	if err != nil {
		return Page{}, err
	}
	defer rows.Close()
	page := Page{Items: []Draft{}}
	for rows.Next() {
		draft, err := scanDraft(rows)
		if err != nil {
			return Page{}, err
		}
		page.Items = append(page.Items, draft)
	}
	if err := rows.Err(); err != nil {
		return Page{}, err
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[limit-1]
		data, err := json.Marshal(cursor{CreatedAt: last.CreatedAt, ID: last.ID})
		if err != nil {
			return Page{}, err
		}
		page.NextCursor = base64.RawURLEncoding.EncodeToString(data)
	}
	return page, tx.Commit(ctx)
}

func (s *Service) UpdateDraft(ctx context.Context, actorID, id uuid.UUID, input UpdateInput, requestID string) (Draft, error) {
	if input.ExpectedConfigVersion < 1 || (input.Slug == nil && input.Title == nil && input.Description == nil) {
		return Draft{}, access.ErrInvalidInput
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Draft{}, err
	}
	defer tx.Rollback(ctx)
	actor, err := access.LockActor(ctx, tx, actorID)
	if err != nil {
		return Draft{}, err
	}
	if !access.CanCreateTournament(actor.Role) {
		return Draft{}, access.ErrForbidden
	}
	draft, err := scanDraft(tx.QueryRow(ctx, `select `+columns+` from tournaments where id=$1 for update`, id))
	if err != nil {
		return Draft{}, err
	}
	if !access.CanManageTournament(actor.Role, actor.ID, draft.OwnerID) {
		return Draft{}, access.ErrNotFound
	}
	if draft.Status != "draft" || draft.ConfigVersion != input.ExpectedConfigVersion {
		return Draft{}, access.ErrConflict
	}
	next := CreateInput{Slug: draft.Slug, Title: draft.Title, Description: draft.Description}
	if input.Slug != nil {
		next.Slug = *input.Slug
	}
	if input.Title != nil {
		next.Title = *input.Title
	}
	if input.Description != nil {
		next.Description = *input.Description
	}
	next, err = normalize(next)
	if err != nil {
		return Draft{}, err
	}
	changed := []string{}
	if next.Slug != draft.Slug {
		changed = append(changed, "slug")
	}
	if next.Title != draft.Title {
		changed = append(changed, "title")
	}
	if next.Description != draft.Description {
		changed = append(changed, "description")
	}
	if len(changed) == 0 {
		return draft, tx.Commit(ctx)
	}
	updated, err := scanDraft(tx.QueryRow(ctx, `update tournaments set slug=$2,title=$3,description=$4,
        config_version=config_version+1,updated_at=clock_timestamp() where id=$1 returning `+columns,
		id, next.Slug, next.Title, next.Description))
	if err != nil {
		return Draft{}, err
	}
	if err := audit.Append(ctx, tx, audit.Event{
		ActorID: &actorID, Action: "tournament.edited", EntityType: "tournament", EntityID: id,
		TournamentID: &id, RequestID: requestID,
		Metadata: map[string]any{"changedFields": changed, "previousConfigVersion": draft.ConfigVersion, "configVersion": updated.ConfigVersion},
	}); err != nil {
		return Draft{}, err
	}
	return updated, tx.Commit(ctx)
}
