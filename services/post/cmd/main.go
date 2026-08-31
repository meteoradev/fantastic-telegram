// @title           Blog API
// @version         1.0
// @description     REST API РґР»СЏ Р±Р»РѕРіР° СЃ Р°РІС‚РѕСЂРёР·Р°С†РёРµР№, СѓРїСЂР°РІР»РµРЅРёРµРј РїРѕР»СЊР·РѕРІР°С‚РµР»СЏРјРё Рё РїРѕСЃС‚Р°РјРё
// @host            localhost:8081
// @BasePath        /
// @schemes         http
// @securityDefinitions.apikey BearerAuth
// @in                        header
// @name                      Authorization
// @description               Р’РІРµРґРёС‚Рµ С‚РѕРєРµРЅ РІ С„РѕСЂРјР°С‚Рµ: Bearer <РІР°С€_С‚РѕРєРµРЅ>

package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/meteoradev/fantastic-telegram/services/post/config"
	"github.com/meteoradev/fantastic-telegram/services/post/internal/repository/postgres"
	o "github.com/meteoradev/fantastic-telegram/services/post/internal/repository/postgres/outbox"
	"github.com/meteoradev/fantastic-telegram/services/post/internal/repository/postgres/post"
	"github.com/meteoradev/fantastic-telegram/services/post/internal/repository/redis"
	"github.com/meteoradev/fantastic-telegram/services/post/internal/service/outbox"
	p "github.com/meteoradev/fantastic-telegram/services/post/internal/service/post"
	"github.com/segmentio/kafka-go"

	_ "github.com/meteoradev/fantastic-telegram/services/post/docs"

	"github.com/meteoradev/fantastic-telegram/services/post/internal/handler"

	gr "github.com/meteoradev/fantastic-telegram/services/post/internal/clients/grpc"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {
	// Logging
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger().Output(zerolog.ConsoleWriter{Out: os.Stdout})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	logger.Info().Msg("Starting the application.")

	// Load cfg
	if _, err := os.Stat("../../.env"); err == nil {
		err := godotenv.Load("../../.env")
		if err != nil {
			logger.Fatal().Err(err).Msg("Fatal error during parse env file.")
		}
	}
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Connect to redis
	rdb, err := redis.RedisConnect(cfg)
	if err != nil {
		logger.Err(err).Str("component", "Redis").Msg("Redis could not connect to db.")
	}
	cache := redis.NewRedisCache(rdb)

	// Connect to DB
	DB, err := postgres.NewPostgresDB(cfg)
	if err != nil {
		logger.Fatal().Err(err).Str("component", "Postgres").Msg("Postgres could not connect to db.")
	}

	// Prepare producer
	brokers := []string{cfg.KafkaHost + ":" + strconv.FormatInt(int64(cfg.KafkaPort), 10)}
	writer := kafka.NewWriter(kafka.WriterConfig{Brokers: brokers, Topic: cfg.KafkaTopic})
	defer func() { _ = writer.Close() }()
	outboxRepo := o.NewOutboxRepository(DB)
	producer := outbox.NewOutboxProducer(outboxRepo, writer, 5*time.Second)
	ctx = logger.WithContext(ctx)
	go producer.Run(ctx)

	logger.Info().Msg("Producer started")
	// Prepare post controller
	postRepo := post.NewPostRepository(DB)
	postSVC := p.NewPostService(postRepo, cache)
	postCtrl := handler.NewPostController(postSVC)

	// Prepare grpc client
	grpcClient, err := gr.NewGRPCClient("user:" + strconv.FormatInt(int64(cfg.GrpcPort), 10))
	if err != nil {
		logger.Fatal().Err(err).Str("component", "GRPC").Msg("GRPC client could not start")
	}

	r := handler.NewRouter(rdb, postCtrl, grpcClient, cfg.ProtectedRPM, cfg.PublicRPM, &logger)

	// Swagger
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:"+strconv.FormatInt(int64(cfg.HTTPPort), 10)+"/swagger/doc.json"),
	))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Start server
	srv := &http.Server{
		Addr:    ":" + strconv.FormatInt(int64(cfg.HTTPPort), 10),
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().
				Err(err).
				Msg("Critical error during starting server")
		}
	}()

	<-ctx.Done()

	// Shutdown
	logger.Info().Msg("Shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal().
			Err(err).
			Msg("Server shutdown failed")
	}
	if err := grpcClient.Conn.Close(); err != nil {
		logger.Fatal().Err(err).Msg("Fatal error during closing GRPC connection.")
	}
	if err := DB.Close(); err != nil {
		logger.Fatal().Err(err).Msg("Fatal error during parse env file.")
	}
	if err := rdb.Close(); err != nil {
		logger.Fatal().Err(err).Msg("Fatal error during parse env file.")
	}
}

