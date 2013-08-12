package plugins

import (
	"../../seabird"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/thoj/go-ircevent"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var country_replacements = map[string]string{
	"US": "USA",
	"United States of America": "USA",
}

type WeatherDay struct {
	Name string `json:"name"`
	Main struct {
		Temp    float32 `json:"temp"`
		TempMin float32 `json:"temp_min"`
		TempMax float32 `json:"temp_max"`
	} `json:"main"`
	Temp struct {
		Min float32 `json:"min"`
		Max float32 `json:"max"`
	} `json:"temp"`
	Sys struct {
		Country string `json:"country"`
	} `json:"sys"`
	Weather []*struct {
		Description string `json:"description"`
	} `json:"weather"`
}

type WeatherResponse struct {
	City struct {
		Name    string `json:"name"`
		Country string `json:"country"`
	} `json:"city"`
	List []*WeatherDay
}

func init() {
	seabird.RegisterPlugin("weather", NewWeatherPlugin)
}

type WeatherPlugin struct {
	Bot *seabird.Bot
}

func NewWeatherPlugin(b *seabird.Bot, c json.RawMessage) {
	p := &WeatherPlugin{b}
	b.RegisterFunction("forecast", p.Forecast)
	b.RegisterFunction("weather", p.Weather)
}

func (p *WeatherPlugin) processDay(d *WeatherDay) error {
	if len(d.Weather) < 1 {
		return errors.New("invalid api response")
	}

	if replacement, ok := country_replacements[d.Sys.Country]; ok {
		d.Sys.Country = replacement
	}

	return nil
}

func (p *WeatherPlugin) weather(loc string) (*WeatherDay, error) {
	var query string = strings.TrimSpace(loc)
	if _, err := strconv.Atoi(query); err == nil {
		// It's a number - append ,USA
		query = query + ",USA"
	}

	resp, err := http.Get(fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?units=imperial&q=%s", url.QueryEscape(query)))
	if err != nil {
		return nil, errors.New("network error")
	}

	weather := WeatherDay{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&weather)
	if err != nil {
		return nil, errors.New("invalid api response")
	}

	if err := p.processDay(&weather); err != nil {
		return nil, err
	}

	return &weather, nil
}

func (p *WeatherPlugin) forecast(loc string, count int) (*WeatherResponse, error) {
	var query string = strings.TrimSpace(loc)
	if _, err := strconv.Atoi(query); err == nil {
		// It's a number - append ,USA
		query = query + ",USA"
	}

	resp, err := http.Get(fmt.Sprintf("http://api.openweathermap.org/data/2.5/forecast/daily?cnt=%d&units=imperial&q=%s", count, url.QueryEscape(query)))
	if err != nil {
		return nil, errors.New("network error")
	}

	weather := WeatherResponse{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&weather)
	if err != nil || len(weather.List) < count {
		return nil, errors.New("invalid api response")
	}

	for _, v := range weather.List {
		if err := p.processDay(v); err != nil {
			return nil, err
		}
	}

	if replacement, ok := country_replacements[weather.City.Country]; ok {
		weather.City.Country = replacement
	}

	return &weather, nil
}

func (p *WeatherPlugin) Forecast(e *irc.Event) {
	weather, err := p.forecast(e.Message, 3)
	if err != nil {
		p.Bot.MentionReply(e, "%s", err.Error())
	}
	p.Bot.MentionReply(e, "3 day forecast for %s, %s.", weather.City.Name, weather.City.Country)
	for _, loc := range weather.List {
		p.Bot.MentionReply(e,
			"High %.2f, Low %.2f, %s.",
			loc.Temp.Max, loc.Temp.Min,
			loc.Weather[0].Description)
	}
}

func (p *WeatherPlugin) Weather(e *irc.Event) {
	loc, err := p.weather(e.Message)
	if err != nil {
		p.Bot.MentionReply(e, "%s", err.Error())
	}
	p.Bot.MentionReply(e,
		"%s, %s. Currently %.1f. High %.2f, Low %.2f, %s.",
		loc.Name, loc.Sys.Country,
		loc.Main.Temp, loc.Main.TempMax, loc.Main.TempMin,
		loc.Weather[0].Description)
}
