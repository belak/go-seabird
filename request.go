package seabird

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	irc "gopkg.in/irc.v3"
)

type Request struct {
	Message *irc.Message

	bot     *Bot
	context context.Context
}

func NewRequest(ctx context.Context, b *Bot, currentNick string, m *irc.Message) *Request {
	ctx = context.WithValue(ctx, contextKeyCurrentNick, currentNick)
	ctx = context.WithValue(ctx, contextKeyRequestID, uuid.New())

	r := &Request{
		m,
		b,
		ctx,
	}

	r.SetTimingMap(make(map[string]*Timing))

	return r
}

func (r *Request) Copy() *Request {
	return &Request{
		r.Message.Copy(),
		r.bot,
		r.context,
	}
}

func (r *Request) Context() context.Context {
	return r.context
}

func (r *Request) GetLogger(name string) *logrus.Entry {
	return CtxLogger(r.context, name).WithField("request", r.ID())
}

func (r *Request) ID() uuid.UUID {
	return CtxRequestID(r.context)
}

func (r *Request) CurrentNick() string {
	return CtxCurrentNick(r.context)
}

// FromChannel checks if this message came from a channel or not.
func (r *Request) FromChannel() bool {
	if len(r.Message.Params) < 1 {
		return false
	}

	// The first param is the target, so if this doesn't match the current nick,
	// the message came from a channel.
	return r.Message.Params[0] != r.CurrentNick()
}
