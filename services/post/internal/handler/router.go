package handler

import (
	"github.com/meteoradev/fantastic-telegram/services/post/internal/middleware"
	"github.com/meteoradev/fantastic-telegram/services/post/internal/port"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

func NewRouter(
	rdb *redis.Client,
	postCtrl *PostController,
	grpcClient port.GRPCClient,
	publicRPM int,
	protectedPRM int,
	logger *zerolog.Logger,
) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RecoveryMiddleware)
	r.Use(middleware.LoggingMiddleware(logger))
	r.Route("/posts", func(r chi.Router) {
		// Public routes
		r.Group(func(r chi.Router) {
			r.Get("/id/{postID}", postCtrl.GetByID)
			r.Get("/title/{title}", postCtrl.GetByTitle)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.NewRateLimitMiddleware(rdb, protectedPRM).Limit)
			r.Use(middleware.NewAuthMiddleware(grpcClient).Authenticate)
			r.Post("/", postCtrl.Create)
			r.Put("/{postID}", postCtrl.Update)
			r.Delete("/{postID}", postCtrl.Delete)
		})

	})

	return r
}

