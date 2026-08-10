package port

import (
	"context"

	"github.com/disdreamq/fantastic-telegram/services/post/internal/domain"
)

type PostRepository interface {
	Create(ctx context.Context, post *domain.Post) (*domain.Post, error)
	GetByID(ctx context.Context, id int64) (*domain.Post, error)
	GetByTitle(ctx context.Context, title string) (*domain.Post, error)
	Update(ctx context.Context, post *domain.Post) error
	UpdateWithValidate(ctx context.Context, currUserID int64, post *domain.Post) error
	Delete(ctx context.Context, id int64) (string, error)
	DeleteWithValidate(ctx context.Context, currUserID, id int64) (string, error)
}
