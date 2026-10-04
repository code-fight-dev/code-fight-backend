package administration

import (
	"context"
	"errors"
	"sort"
	"strings"
	"unicode/utf8"

	"code-fight-backend/internal/access"
	"code-fight-backend/internal/audit"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type database interface {
	Begin(context.Context) (pgx.Tx, error)
}

type Service struct{ db database }

func NewService(db database) *Service { return &Service{db: db} }

type User struct {
	ID          uuid.UUID   `json:"id"`
	Username    string      `json:"username"`
	Role        access.Role `json:"role"`
	RoleVersion int64       `json:"roleVersion"`
}

type ChangeRoleInput struct {
	Role                access.Role `json:"role"`
	ExpectedRoleVersion int64       `json:"expectedRoleVersion"`
	Reason              string      `json:"reason"`
}

func readUser(ctx context.Context, tx pgx.Tx, id uuid.UUID, lock bool) (User, error) {
	query := `select id, username, role, role_version from users where id=$1`
	if lock {
		query += ` for update`
	}
	var user User
	err := tx.QueryRow(ctx, query, id).Scan(&user.ID, &user.Username, &user.Role, &user.RoleVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, access.ErrNotFound
	}
	return user, err
}

func (s *Service) GetUser(ctx context.Context, actorID, targetID uuid.UUID) (User, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)
	actor, err := access.LockActor(ctx, tx, actorID)
	if err != nil {
		return User{}, err
	}
	if actor.Role != access.Admin {
		return User{}, access.ErrForbidden
	}
	user, err := readUser(ctx, tx, targetID, false)
	if err != nil {
		return User{}, err
	}
	return user, tx.Commit(ctx)
}

func (s *Service) ChangeRole(ctx context.Context, actorID, targetID uuid.UUID, input ChangeRoleInput, requestID string) (User, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if actorID == uuid.Nil || targetID == uuid.Nil || requestID == "" ||
		(input.Role != access.User && input.Role != access.Organizer) ||
		input.ExpectedRoleVersion < 1 || !validReason(input.Reason) {
		return User{}, access.ErrInvalidInput
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)

	ids := []uuid.UUID{actorID}
	if targetID != actorID {
		ids = append(ids, targetID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	locked := make(map[uuid.UUID]User, len(ids))
	for _, id := range ids {
		user, err := readUser(ctx, tx, id, true)
		if err != nil {
			return User{}, err
		}
		locked[id] = user
	}
	if locked[actorID].Role != access.Admin {
		return User{}, access.ErrForbidden
	}
	target := locked[targetID]
	if target.Role == access.Admin {
		return User{}, access.ErrForbidden
	}
	if target.RoleVersion != input.ExpectedRoleVersion {
		return User{}, access.ErrConflict
	}
	if target.Role == input.Role {
		return target, tx.Commit(ctx)
	}

	previousRole := target.Role
	if err := tx.QueryRow(ctx, `update users set role=$2, role_version=role_version+1,
        updated_at=clock_timestamp() where id=$1 returning role, role_version`,
		targetID, input.Role).Scan(&target.Role, &target.RoleVersion); err != nil {
		return User{}, err
	}
	action := "role.granted"
	if input.Role == access.User {
		action = "role.revoked"
	}
	if err := audit.Append(ctx, tx, audit.Event{
		ActorID: &actorID, Action: action, EntityType: "user", EntityID: targetID, RequestID: requestID,
		Metadata: map[string]any{"previousRole": previousRole, "role": target.Role,
			"roleVersion": target.RoleVersion, "reason": input.Reason},
	}); err != nil {
		return User{}, err
	}
	return target, tx.Commit(ctx)
}

var ErrAdminAlreadyExists = errors.New("an admin already exists; bootstrap is disabled")

func validReason(reason string) bool {
	return utf8.RuneCountInString(reason) >= 1 && utf8.RuneCountInString(reason) <= 500
}

func (s *Service) BootstrapAdmin(ctx context.Context, targetID uuid.UUID, reason, requestID string) (User, error) {
	reason = strings.TrimSpace(reason)
	if targetID == uuid.Nil || !validReason(reason) || requestID == "" {
		return User{}, access.ErrInvalidInput
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(172936, 1)`); err != nil {
		return User{}, err
	}
	var exists bool
	if err := tx.QueryRow(ctx, `select exists(select 1 from users where role='admin')`).Scan(&exists); err != nil {
		return User{}, err
	}
	if exists {
		return User{}, ErrAdminAlreadyExists
	}
	target, err := readUser(ctx, tx, targetID, true)
	if err != nil {
		return User{}, err
	}
	previousRole := target.Role
	if err := tx.QueryRow(ctx, `update users set role='admin', role_version=role_version+1,
        updated_at=clock_timestamp() where id=$1 returning role, role_version`, targetID).
		Scan(&target.Role, &target.RoleVersion); err != nil {
		return User{}, err
	}
	if err := audit.Append(ctx, tx, audit.Event{
		Action: "role.admin_bootstrapped", EntityType: "user", EntityID: targetID, RequestID: requestID,
		Metadata: map[string]any{"previousRole": previousRole, "role": target.Role,
			"roleVersion": target.RoleVersion, "reason": reason, "trigger": "admin-bootstrap-cli"},
	}); err != nil {
		return User{}, err
	}
	return target, tx.Commit(ctx)
}
