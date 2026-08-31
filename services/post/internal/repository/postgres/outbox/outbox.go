package outbox

import (
	"context"
	"time"

	"github.com/meteoradev/fantastic-telegram/services/post/internal/domain"
	"github.com/jmoiron/sqlx"
)

type dbOutboxPost struct {
	ID        int64     `db:"id"`
	Payload   string    `db:"payload"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (d *dbOutboxPost) toDomain() *domain.OutboxPost {
	return &domain.OutboxPost{
		ID:      d.ID,
		Payload: d.Payload,
		Status:  d.Status,
	}
}

type OutboxRepository struct {
	db *sqlx.DB
}

func NewOutboxRepository(db *sqlx.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) GetPosts(ctx context.Context) ([]*domain.OutboxPost, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	txCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	query := `
        SELECT * FROM posts_outbox
        WHERE status != 'PROCESSED'
        ORDER BY created_at
        LIMIT 10
        FOR UPDATE SKIP LOCKED
    `
	var oPosts []dbOutboxPost
	if err = tx.SelectContext(txCtx, &oPosts, query); err != nil {
		return nil, err
	}

	res := make([]*domain.OutboxPost, 0, len(oPosts))
	for i := range oPosts {
		res = append(res, oPosts[i].toDomain())
	}
	return res, nil
}

func (r *OutboxRepository) UpdatePosts(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	txCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	query, args, err := sqlx.In(`
        UPDATE posts_outbox
        SET updated_at = NOW(), status = 'PROCESSED'
        WHERE id IN (?)
    `, ids)
	if err != nil {
		return err
	}
	query = tx.Rebind(query)

	_, err = tx.ExecContext(txCtx, query, args...)
	if err != nil {
		return err
	}

	return tx.Commit()
}

