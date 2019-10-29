package seabird

import (
	"context"
	"time"

	irc "gopkg.in/irc.v3"
)

type contextKey string

const (
	timingKey contextKey = "context: timing"
)

type Timing struct {
	Start time.Time
	End   time.Time
}

func (t *Timing) Done() {
	t.End = time.Now()
}

func (t *Timing) Elapsed() time.Duration {
	return t.End.Sub(t.Start)
}

type Request struct {
	Message *irc.Message
	Context context.Context
}

func NewRequest(m *irc.Message) *Request {
	r := &Request{
		m,
		context.TODO(),
	}

	r.SetTimingMap(make(map[string]*Timing))

	return r
}

func (r *Request) Copy() *Request {
	return &Request{
		r.Message.Copy(),
		r.Context,
	}
}

func (r *Request) TimingMap() map[string]*Timing {
	return r.Context.Value(timingKey).(map[string]*Timing)
}

func (r *Request) SetTimingMap(tc map[string]*Timing) {
	r.Context = context.WithValue(r.Context, timingKey, tc)
}

func (r *Request) AddTiming(name string, t *Timing) {
}

func (r *Request) Timer(event string) *Timing {
	timer := &Timing{
		Start: time.Now(),
	}

	ctx := r.TimingMap()
	ctx[event] = timer

	return timer
}
