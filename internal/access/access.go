package access

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Role string

const (
	User      Role = "user"
	Organizer Role = "organizer"
	Admin     Role = "admin"
)

var (
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("resource not found")
	ErrInvalidInput = errors.New("invalid request")
	ErrConflict     = errors.New("resource version changed")
)

func (r Role) Valid() bool { return r == User || r == Organizer || r == Admin }

func CanCreateTournament(role Role) bool { return role == Organizer || role == Admin }

func CanManageTournament(role Role, actorID, ownerID uuid.UUID) bool {
	return actorID != uuid.Nil && (role == Admin || (role == Organizer && actorID == ownerID))
}

type Actor struct {
	ID          uuid.UUID
	Role        Role
	RoleVersion int64
}

func LockActor(ctx context.Context, tx pgx.Tx, id uuid.UUID) (Actor, error) {
	var actor Actor
	err := tx.QueryRow(ctx, `select id, role, role_version from users where id=$1 for share`, id).
		Scan(&actor.ID, &actor.Role, &actor.RoleVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return Actor{}, ErrForbidden
	}
	return actor, err
}
