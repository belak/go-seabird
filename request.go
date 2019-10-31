package seabird

import (
	"context"
	"time"

	"github.com/influxdata/influxdb1-client/v2"

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

func (r *Request) Log(bot *Bot) {
	timings := r.TimingMap()

	fields := make(map[string]interface{})

	completeEvents := 0
	incompleteEvents := 0

	for _, timing := range timings {
		keyBase := timing.Title
		fields[keyBase+"-start"] = timing.Start.UnixNano()
		if !timing.Completed {
			incompleteEvents += 1
			continue
		}

		fields[keyBase+"-end"] = timing.End.UnixNano()
		fields[keyBase+"-elapsed"] = timing.Elapsed().Nanoseconds()

		completeEvents += 1
	}

	fields["complete-events"] = completeEvents
	fields["incomplete-events"] = incompleteEvents

	now := time.Now()

	point, err := client.NewPoint("request_timing", map[string]string{}, fields, now)
	if err != nil {
		bot.log.Warning("Error: ", err.Error())
		return
	}

	if bot.points.Len() > bot.influxDbConfig.BufferSize {
		bot.log.Warning("Too many datapoints to buffer. Refusing to buffer more until datapoints are submitted.")
	}

	bot.points.PushBack(point)

	// TODO(jsvana): break this out into a separate thread otherwise datapoints won't get
	// submitted regularly
	if now.Sub(bot.lastPointSubmit).Milliseconds() < bot.influxDbConfig.SubmitIntervalMs {
		return
	}

	batch, _ := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  bot.influxDbConfig.Database,
		Precision: bot.influxDbConfig.Precision,
	})

	for bot.points.Len() > 0 {
		batch.AddPoint(bot.points.Remove(bot.points.Front()).(*client.Point))
	}

	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     bot.influxDbConfig.Url,
		Username: bot.influxDbConfig.Username,
		Password: bot.influxDbConfig.Password,
	})
	if err != nil {
		bot.log.Warning("Error creating InfluxDB Client: ", err.Error())
		return
	}
	defer c.Close()

	err = c.Write(batch)
	if err != nil {
		bot.log.Warning("Error: ", err.Error())
	}

	bot.lastPointSubmit = now
}
