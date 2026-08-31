package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/meteoradev/fantastic-telegram/services/post/internal/port"
)

type AuthMiddleware struct {
	grpcClient port.GRPCClient
}

func NewAuthMiddleware(grpcClient port.GRPCClient) *AuthMiddleware {
	return &AuthMiddleware{
		grpcClient: grpcClient,
	}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
			return
		}
		token := parts[1]
		claims, err := m.grpcClient.ValidateToken(r.Context(), token)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), "userID", claims.UserID)
		ctx = context.WithValue(ctx, "email", claims.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

