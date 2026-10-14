package administration

import (
	"context"
	"errors"
	"testing"

	"code-fight-backend/internal/access"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type rejectDatabase struct{ t *testing.T }

func (db rejectDatabase) Begin(context.Context) (pgx.Tx, error) {
	db.t.Error("invalid reason reached the database")
	return nil, errors.New("unexpected database call")
}

func TestNULReasonRejectedBeforeTransaction(t *testing.T) {
	service := NewService(rejectDatabase{t})
	ctx := context.Background()
	input := ChangeRoleInput{Role: access.Organizer, ExpectedRoleVersion: 1, Reason: "invalid\x00reason"}
	if _, err := service.ChangeRole(ctx, uuid.New(), uuid.New(), input, uuid.NewString()); !errors.Is(err, access.ErrInvalidInput) {
		t.Fatalf("role change error = %v", err)
	}
	if _, err := service.BootstrapAdmin(ctx, uuid.New(), input.Reason, uuid.NewString()); !errors.Is(err, access.ErrInvalidInput) {
		t.Fatalf("bootstrap error = %v", err)
	}
}
