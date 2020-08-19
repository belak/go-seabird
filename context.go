package seabird

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/belak/go-seabird/internal"
)

const (
	contextKeyBot    = internal.ContextKey("seabird-bot")
	contextKeyLogger = internal.ContextKey("seabird-logger-entry")

	contextKeyCurrentNick = internal.ContextKey("seabird-current-nick")
	contextKeyRequestID   = internal.ContextKey("seabird-request-id")
)

func withSeabirdValues(ctx context.Context, b *Bot, log *logrus.Entry) context.Context {
	ctx = context.WithValue(ctx, contextKeyBot, b)
	ctx = context.WithValue(ctx, contextKeyLogger, log)

	return ctx
}

func CtxBot(ctx context.Context) *Bot {
	return ctx.Value(contextKeyBot).(*Bot)
}

func CtxLogger(ctx context.Context, name string) *logrus.Entry {
	logger := ctx.Value(contextKeyLogger).(*logrus.Entry)

	return logger.WithField("category", name)
}

func CtxCurrentNick(ctx context.Context) string {
	return ctx.Value(contextKeyCurrentNick).(string)
}

func CtxRequestID(ctx context.Context) uuid.UUID {
	return ctx.Value(contextKeyRequestID).(uuid.UUID)
}
