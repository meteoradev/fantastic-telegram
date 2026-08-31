package outbox

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/meteoradev/fantastic-telegram/services/post/internal/domain"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

type OutboxRepository interface {
	GetPosts(context.Context) ([]*domain.OutboxPost, error)
	UpdatePosts(context.Context, []int64) error
}

type Producer struct {
	outboxRepo OutboxRepository
	producer   *kafka.Writer
	interval   time.Duration
	ticker     *time.Ticker
}

func NewOutboxProducer(outboxRepo OutboxRepository, producer *kafka.Writer, interval time.Duration) *Producer {
	return &Producer{
		outboxRepo: outboxRepo,
		producer:   producer,
		interval:   interval,
	}
}

func (p *Producer) Run(ctx context.Context) {
	p.ticker = time.NewTicker(p.interval)
	defer p.ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Print("Outbox publisher stopped")
			return
		case <-p.ticker.C:
			p.Process(ctx)
		}
	}
}

func (s *Producer) Process(ctx context.Context) {
	logger := log.Ctx(ctx)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	posts, err := s.outboxRepo.GetPosts(ctx)
	if len(posts) == 0 {
		return
	}
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to send posts to kafka")
		return
	}
	msgs := make([]kafka.Message, 0, len(posts))
	ids := make([]int64, 0, len(posts))
	for _, post := range posts {
		msg := kafka.Message{
			Value: []byte(post.Payload),
		}
		msgs = append(msgs, msg)
		ids = append(ids, post.ID)
	}
	var b strings.Builder
	for i, id := range ids {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(strconv.FormatInt(id, 10))
	}

	if err = s.producer.WriteMessages(ctx, msgs...); err != nil {
		logger.Error().
			Err(err).
			Str("IDs", b.String()).
			Msg("Failed to send posts to kafka")
		return
	}

	if err := s.outboxRepo.UpdatePosts(ctx, ids); err != nil {
		logger.Error().
			Err(err).
			Str("IDs", b.String()).
			Msg("Failed to update posts in outbox db")
	}

	logger.Info().
		Str("IDs", b.String()).
		Msg("Send posts to kafka")
}

