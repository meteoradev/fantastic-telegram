package port

import (
	"context"

	"github.com/meteoradev/fantastic-telegram/services/post/internal/domain"
)

type GRPCClient interface {
	ValidateToken(ctx context.Context, token string) (*domain.Claims, error)
}

