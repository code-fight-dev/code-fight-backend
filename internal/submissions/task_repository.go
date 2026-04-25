package submissions

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PGTaskRepository struct {
	db *pgxpool.Pool
}

func NewPGTaskRepository(db *pgxpool.Pool) *PGTaskRepository {
	return &PGTaskRepository{db: db}
}

func (r *PGTaskRepository) GetByID(ctx context.Context, taskID uuid.UUID) (*Task, error) {
	const q = `
select id, is_active
from tasks
where id = $1
`

	var t Task
	err := r.db.QueryRow(ctx, q, taskID).Scan(
		&t.ID,
		&t.IsActive,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &t, nil
}

func (r *PGTaskRepository) IsLanguageAllowed(
	ctx context.Context,
	taskID uuid.UUID,
	language, languageVersion string,
) (bool, error) {
	const q = `
select exists(
	select 1
	from task_languages
	where task_id = $1
	  and language = $2
	  and language_version = $3
	  and is_enabled = true
)
`

	var ok bool
	if err := r.db.QueryRow(ctx, q, taskID, language, languageVersion).Scan(&ok); err != nil {
		return false, err
	}

	return ok, nil
}