package seabird

import (
	"context"

	"github.com/belak/go-seabird/internal"
	"github.com/sirupsen/logrus"
)

const (
	contextKeyBot    = internal.ContextKey("seabird-bot")
	contextKeyLogger = internal.ContextKey("seabird-logger-entry")

	// TODO: set this
	contextKeyCurrentNick = internal.ContextKey("seabird-current-nick")
)

func withSeabirdValues(ctx context.Context, b *Bot, log *logrus.Entry) context.Context {
	ctx = context.WithValue(ctx, contextKeyBot, b)
	ctx = context.WithValue(ctx, contextKeyLogger, log)

	return ctx
}

func CtxBot(ctx context.Context) *Bot {
	return ctx.Value(contextKeyBot).(*Bot)
}

func CtxLogger(ctx context.Context) *logrus.Entry {
	return ctx.Value(contextKeyLogger).(*logrus.Entry)
}

func CtxCurrentNick(ctx context.Context) string {
	return ctx.Value(contextKeyCurrentNick).(string)
}
