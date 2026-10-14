package tournament

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
	db.t.Error("invalid input reached the database")
	return nil, errors.New("unexpected database call")
}

func TestNULRejectedBeforeTransaction(t *testing.T) {
	service := NewService(rejectDatabase{t})
	for _, field := range []string{"slug", "title", "description"} {
		t.Run(field, func(t *testing.T) {
			create := CreateInput{Slug: "valid-cup", Title: "Valid title"}
			update := UpdateInput{ExpectedConfigVersion: 1}
			value := "invalid\x00value"
			switch field {
			case "slug":
				create.Slug, update.Slug = value, &value
			case "title":
				create.Title, update.Title = value, &value
			case "description":
				create.Description, update.Description = value, &value
			}
			if _, err := service.CreateDraft(context.Background(), uuid.New(), create, uuid.NewString()); !errors.Is(err, access.ErrInvalidInput) {
				t.Fatalf("create error = %v", err)
			}
			if _, err := service.UpdateDraft(context.Background(), uuid.New(), uuid.New(), update, uuid.NewString()); !errors.Is(err, access.ErrInvalidInput) {
				t.Fatalf("update error = %v", err)
			}
		})
	}
}
