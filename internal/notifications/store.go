package notifications

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const ChangedEvent = "notifications.changed"

type Item struct {
	ID         uuid.UUID  `json:"id"`
	Kind       string     `json:"kind"`
	Title      string     `json:"title"`
	Message    string     `json:"message"`
	Href       string     `json:"href"`
	DeadlineAt *time.Time `json:"deadlineAt"`
	CreatedAt  time.Time  `json:"createdAt"`
	ReadAt     *time.Time `json:"readAt"`
}

type Page struct {
	Items      []Item     `json:"items"`
	Unread     int        `json:"unread"`
	NextCursor *uuid.UUID `json:"nextCursor"`
}

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store { return &Store{db: db} }

// Insert writes a notification in the same transaction as its originating action.
func Insert(ctx context.Context, tx pgx.Tx, userID uuid.UUID, item Item) error {
	_, err := tx.Exec(ctx, `insert into notifications (user_id,kind,title,message,href,deadline_at)
		values ($1,$2,$3,$4,$5,$6)`, userID, item.Kind, item.Title, item.Message, item.Href, item.DeadlineAt)
	return err
}

func (s *Store) List(ctx context.Context, userID uuid.UUID, before *uuid.UUID) (Page, error) {
	const pageSize = 20
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Page{}, err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	var page Page
	if err := tx.QueryRow(ctx, `select count(*) from notifications where user_id=$1 and read_at is null`, userID).
		Scan(&page.Unread); err != nil {
		return Page{}, err
	}
	rows, err := tx.Query(ctx, `select id,kind,title,message,href,deadline_at,created_at,read_at
		from notifications
		where user_id=$1 and ($2::uuid is null or (created_at,id) < (
			select created_at,id from notifications where user_id=$1 and id=$2
		))
		order by created_at desc,id desc limit $3`, userID, before, pageSize+1)
	if err != nil {
		return Page{}, err
	}
	page.Items, err = pgx.CollectRows(rows, pgx.RowToStructByName[Item])
	if err != nil {
		return Page{}, err
	}
	if len(page.Items) > pageSize {
		page.Items = page.Items[:pageSize]
		page.NextCursor = &page.Items[pageSize-1].ID
	}
	return page, tx.Commit(ctx)
}

func (s *Store) MarkRead(ctx context.Context, userID uuid.UUID, id *uuid.UUID) error {
	_, err := s.db.Exec(ctx, `update notifications set read_at=clock_timestamp()
		where user_id=$1 and ($2::uuid is null or id=$2) and read_at is null`, userID, id)
	return err
}
