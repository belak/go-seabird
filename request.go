package seabird

import (
	"context"
	"fmt"
	"time"

	irc "gopkg.in/irc.v3"
)

type Timing struct {
	Start time.Time
	End   time.Time
}

func (d *Timing) Done() {
	d.End = time.Now()
}

func (d *Timing) Elapsed() time.Duration {
	return d.End.Sub(d.Start)
}

type Request struct {
	Message *irc.Message
	Context context.Context
}

func (r *Request) Copy() *Request {
	return &Request{
		r.Message.Copy(),
		r.Context,
	}
}

func (r *Request) SetValue(key string, value interface{}) {
}

func (r *Request) Timer(event string) *Timing {
	timer := &Timing{
		Start: time.Now(),
	}

	// TODO: nest this within a namespace
	r.SetValue(event, timer)

	return timer
}
