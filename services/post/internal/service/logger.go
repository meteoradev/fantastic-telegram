package service

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// logWithTrace creates a zerolog event with trace_id from context.
func logWithTrace(ctx context.Context, level zerolog.Level) *zerolog.Event {
	logger := log.Ctx(ctx)
	if logger == nil {
		logger = &log.Logger
	}
	c := logger.With()
	if tid, ok := ctx.Value("trace_id").(string); ok {
		c.Str("trace_id", tid)
	}
	l := c.Logger()
	switch level {
	case zerolog.DebugLevel:
		return l.Debug()
	case zerolog.InfoLevel:
		return l.Info()
	case zerolog.WarnLevel:
		return l.Warn()
	case zerolog.ErrorLevel:
		return l.Error()
	case zerolog.FatalLevel:
		return l.Fatal()
	case zerolog.PanicLevel:
		return l.Panic()
	default:
		return l.Info()
	}
}

// LogError logs an error with optional key-value fields.
func LogError(ctx context.Context, msg string, fields ...any) {
	ev := logWithTrace(ctx, zerolog.ErrorLevel)
	addFields(ev, fields...)
	ev.Msg(msg)
}

// LogInfo logs an info with optional key-value fields.
func LogInfo(ctx context.Context, msg string, fields ...any) {
	ev := logWithTrace(ctx, zerolog.InfoLevel)
	addFields(ev, fields...)
	ev.Msg(msg)
}

// LogDebug logs a debug with optional key-value fields.
func LogDebug(ctx context.Context, msg string, fields ...any) {
	ev := logWithTrace(ctx, zerolog.DebugLevel)
	addFields(ev, fields...)
	ev.Msg(msg)
}

func addFields(ev *zerolog.Event, fields ...any) {
	for i := 0; i+1 < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue
		}
		switch v := fields[i+1].(type) {
		case string:
			ev.Str(key, v)
		case int:
			ev.Int(key, v)
		case int64:
			ev.Int64(key, v)
		case float64:
			ev.Float64(key, v)
		case error:
			ev.Err(v)
		default:
			ev.Interface(key, v)
		}
	}
}
