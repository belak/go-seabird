package plugins

import (
	"strings"
	"time"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
	"github.com/belak/seabird/plugins/infoproviders"
)

func init() {
	bot.RegisterPlugin("forecast", NewForecastPlugin)
}

type ForecastPlugin struct {
	w *infoproviders.WeatherProvider
}

func NewForecastPlugin(b *bot.Bot, m *mux.CommandMux, weather *infoproviders.WeatherProvider) error {
	p := &ForecastPlugin{weather}

	m.Event("weather", p.Weather, &mux.HelpInfo{
		"<location>",
		"Retrieves current weather for given location",
	})
	m.Event("forecast", p.Forecast, &mux.HelpInfo{
		"<location>",
		"Retrieves three-day forecast for given location",
	})

	return nil
}

func (p *ForecastPlugin) Forecast(c *irc.Client, e *irc.Event) {
	loc, err := p.w.GetLocation(strings.TrimSpace(e.Trailing()))
	if err != nil {
		c.MentionReply(e, "%s", err.Error())
		return
	}

	fc, err := p.w.ForecastQuery(loc.Coords)
	if err != nil {
		c.MentionReply(e, "%s", err.Error())
		return
	}

	c.MentionReply(e, "3 day forecast for %s.", loc.Address)
	for _, block := range fc.Daily.Data[1:4] {
		day := time.Unix(block.Time, 0).Weekday()

		c.MentionReply(e,
			"%s: High %.2f, Low %.2f, Humidity %.f%%. %s",
			day,
			block.TemperatureMax,
			block.TemperatureMin,
			block.Humidity*100,
			block.Summary)
	}
}

func (p *ForecastPlugin) Weather(c *irc.Client, e *irc.Event) {
	loc, err := p.w.GetLocation(strings.TrimSpace(e.Trailing()))
	if err != nil {
		c.MentionReply(e, "%s", err.Error())
		return
	}

	fc, err := p.w.ForecastQuery(loc.Coords)
	if err != nil {
		c.MentionReply(e, "%s", err.Error())
		return
	}

	today := fc.Daily.Data[0]
	c.MentionReply(e,
		"%s. Currently %.1f. High %.2f, Low %.2f, Humidity %.f%%. %s.",
		loc.Address,
		fc.Currently.Temperature,
		today.TemperatureMax,
		today.TemperatureMin,
		fc.Currently.Humidity*100,
		fc.Currently.Summary)
}
