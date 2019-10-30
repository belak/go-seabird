package seabird

import (
	"context"
	"sort"
	"time"

	"github.com/sirupsen/logrus"

	irc "gopkg.in/irc.v3"
)

type contextKey string

const timingKey = contextKey("context: timing")

type Timing struct {
	Title     string
	Start     time.Time
	End       time.Time
	Completed bool
}

func (t *Timing) Done() {
	t.End = time.Now()
	t.Completed = true
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
		Title:     event,
		Start:     time.Now(),
		Completed: false,
	}

	ctx := r.TimingMap()
	ctx[event] = timer

	return timer
}

func (r *Request) LogTimings(logger *logrus.Entry) {
	timings := r.TimingMap()

	sortedTimings := make([]*Timing, 0, len(timings))
	for _, timing := range timings {
		sortedTimings = append(sortedTimings, timing)
	}

	sort.Slice(sortedTimings, func(i, j int) bool {
		return sortedTimings[i].Start.Before(sortedTimings[j].Start)
	})

	logger.Debug("Request timing:")
	for _, timing := range sortedTimings {
		if !timing.Completed {
			logger.Debugf("%s: [started:%d] [not completed]", timing.Title, timing.Start.UnixNano())
			continue
		}

		logger.Debugf("%s: [start:%d] [duration:%s]", timing.Title, timing.Start.UnixNano(), timing.Elapsed().String())
	}
}
