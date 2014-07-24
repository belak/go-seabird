package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	seabird ".."
	"github.com/thoj/go-ircevent"
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

// Grabs a lat and lon from a location
type WeatherLocationResponse struct {
	Results []struct {
		Address  string `json:"formatted_address"`
		Geometry struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lon float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
	} `json:"results"`
	Status string `json:"status"`
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
	query := strings.TrimSpace(loc)
	if len(query) == 0 {
		return nil, errors.New("missing location")
	}

	l, err := p.locationQuery(query)
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(
		fmt.Sprintf(
			"http://api.openweathermap.org/data/2.5/weather?units=imperial&lat=%.4f&lon=%.4f",
			l.Results[0].Geometry.Location.Lat,
			l.Results[0].Geometry.Location.Lon,
		),
	)

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

func (p *WeatherPlugin) locationQuery(name string) (*WeatherLocationResponse, error) {
	v := url.Values{}
	v.Set("address", name)
	v.Set("sensor", "false")

	u, _ := url.Parse("http://maps.googleapis.com/maps/api/geocode/json")
	u.RawQuery = v.Encode()

	r, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}

	loc := WeatherLocationResponse{}
	dec := json.NewDecoder(r.Body)
	defer r.Body.Close()
	dec.Decode(&loc)

	if len(loc.Results) == 0 {
		return nil, errors.New("No location results found")
	} else if len(loc.Results) > 1 {
		// TODO: display results
		return nil, errors.New("More than 1 result")
	}

	return &loc, nil
}

func (p *WeatherPlugin) forecast(loc string, count int) (*WeatherResponse, error) {
	query := strings.TrimSpace(loc)
	if len(query) == 0 {
		return nil, errors.New("missing location")
	}

	l, err := p.locationQuery(query)
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(
		fmt.Sprintf(
			"http://api.openweathermap.org/data/2.5/forecast/daily?cnt=%d&units=imperial&lat=%.4f&lon=%.4f",
			count,
			l.Results[0].Geometry.Location.Lat,
			l.Results[0].Geometry.Location.Lon,
		),
	)

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
	weather, err := p.forecast(e.Message(), 3)
	if err != nil {
		p.Bot.MentionReply(e, "%s", err.Error())
		return
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
	loc, err := p.weather(e.Message())
	if err != nil {
		p.Bot.MentionReply(e, "%s", err.Error())
		return
	}
	p.Bot.MentionReply(e,
		"%s, %s. Currently %.1f. High %.2f, Low %.2f, %s.",
		loc.Name, loc.Sys.Country,
		loc.Main.Temp, loc.Main.TempMax, loc.Main.TempMin,
		loc.Weather[0].Description)
}
