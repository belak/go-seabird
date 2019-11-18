package seabird

import (
	"context"
	"time"

	client "github.com/influxdata/influxdb1-client/v2"

	"github.com/belak/go-seabird/internal"
)

const timingKey = internal.ContextKey("seabird-timing-map")

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

func (r *Request) TimingMap() map[string]*Timing {
	return r.context.Value(timingKey).(map[string]*Timing)
}

func (r *Request) SetTimingMap(tc map[string]*Timing) {
	r.context = context.WithValue(r.context, timingKey, tc)
}

func (r *Request) Timer(event string) *Timing {
	timer := &Timing{
		Title: event,
		Start: time.Now(),
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
			incompleteEvents++
			continue
		}

		fields[keyBase+"-end"] = timing.End.UnixNano()
		fields[keyBase+"-elapsed"] = timing.Elapsed().Nanoseconds()

		completeEvents++
	}

	fields["complete-events"] = completeEvents
	fields["incomplete-events"] = incompleteEvents

	now := time.Now()

	point, err := client.NewPoint("request_timing", make(map[string]string), fields, now)
	if err != nil {
		bot.log.Warning("Error creating a new InfluxDB datapoint: ", err.Error())
		return
	}

	// Ensure that we don't block the bot by using a blocking insert. Instead, drop
	// requests as necessary.
	select {
	case bot.points <- point:
		return
	default:
		// Only log dropped datapoints if it's enabled
		if bot.influxDbConfig.Enabled {
			bot.log.Warning("InfluxDB datapoint queue is full, dropping datapoint")
		}
	}
}