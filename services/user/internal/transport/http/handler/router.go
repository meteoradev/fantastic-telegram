package handler

import (
	"time"

	"github.com/meteoradev/fantastic-telegram/services/user/internal/infra/jwt"
	"github.com/meteoradev/fantastic-telegram/services/user/internal/transport/http/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

func NewRouter(
	rdb *redis.Client,
	userCtrl *UserController,
	authCtrl *AuthController,
	secret string,
	expiry time.Duration,
	PublicRPM int,
	ProtectedPRM int,
	logger zerolog.Logger,
) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RecoveryMiddleware)
	r.Use(middleware.LoggingMiddleware(logger))

	// Public routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.NewRateLimitMiddleware(rdb, PublicRPM).Limit)
		r.Post("/register", userCtrl.Create)
		r.Post("/login", authCtrl.Login)
		r.Get("/users/{userID}", userCtrl.GetByID)
		r.Get("/users/email/{email}", userCtrl.GetByEmail)
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.NewRateLimitMiddleware(rdb, ProtectedPRM).Limit)
		r.Use(middleware.NewAuthMiddleware(jwt.NewProvider(secret, expiry)).Authenticate)

		r.Route("/users", func(r chi.Router) {
			r.Put("/{userID}", userCtrl.Update)
			r.Delete("/{userID}", userCtrl.Delete)
		})
	})
	return r
}

