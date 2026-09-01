package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/meteoradev/fantastic-telegram/notification/config"
	k "github.com/meteoradev/fantastic-telegram/notification/internal/kafka"
	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
)

func main() {
	// Logging
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger().Output(zerolog.ConsoleWriter{Out: os.Stdout})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	logger.Info().Msg("Starting the application.")

	// Load config
	if _, err := os.Stat("../../.env"); err == nil {
		err := godotenv.Load("../../.env")
		if err != nil {
			logger.Fatal().Err(err).Msg("Fatal error during parse env file.")
		}
	}
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Prepare kafka consumer
	ctx = logger.WithContext(ctx)
	brokers := []string{cfg.KafkaHost + ":" + strconv.FormatInt(int64(cfg.KafkaPort), 10)}
	r := kafka.NewReader(kafka.ReaderConfig{Brokers: brokers, GroupID: "notification", Topic: cfg.KafkaTopic, StartOffset: kafka.FirstOffset})
	consumer := k.NewConsumer(r)
	go consumer.Run(ctx)

	<-ctx.Done()

	// Shutdown
	logger.Info().Msg("Shutting down")

}
