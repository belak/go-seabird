package plugins

import (
	"../../seabird"

	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/thoj/go-ircevent"
)

// Basic structures from https://github.com/mlbright/forecast/blob/master/v2/forecast.go
// TODO: cleanup
type DataPoint struct {
	Time                   float64
	Summary                string
	Icon                   string
	SunriseTime            float64
	SunsetTime             float64
	PrecipIntensity        float64
	PrecipIntensityMax     float64
	PrecipIntensityMaxTime float64
	PrecipProbability      float64
	PrecipType             string
	PrecipAccumulation     float64
	Temperature            float64
	TemperatureMin         float64
	TemperatureMinTime     float64
	TemperatureMax         float64
	TemperatureMaxTime     float64
	DewPoint               float64
	WindSpeed              float64
	WindBearing            float64
	CloudCover             float64
	Humidity               float64
	Pressure               float64
	Visibility             float64
	Ozone                  float64
}

type DataBlock struct {
	Summary string
	Icon    string
	Data    []DataPoint
}

type alert struct {
	Title   string
	Expires float64
	URI     string
}

type Flags struct {
	DarkSkyUnavailable string
	DarkSkyStations    []string
	DataPointStations  []string
	ISDStations        []string
	LAMPStations       []string
	METARStations      []string
	METNOLicense       string
	Sources            []string
	Units              string
}

type ForecastResponse struct {
	Latitude  float64
	Longitude float64
	Timezone  string
	Offset    float64
	Currently DataPoint
	Minutely  DataBlock
	Hourly    DataBlock
	Daily     DataBlock
	Alerts    []alert
	Flags     Flags
	APICalls  int
}

// Grabs a lat and lon from a location
type LocationResponse struct {
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

func (p *ForecastPlugin) locationQuery(name string) (*LocationResponse, error) {
	v := url.Values{}
	v.Set("address", name)
	v.Set("sensor", "false")

	u, _ := url.Parse("http://maps.googleapis.com/maps/api/geocode/json")
	u.RawQuery = v.Encode()

	r, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}

	loc := LocationResponse{}
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

func (p *ForecastPlugin) forecastQuery(m string) (*ForecastResponse, *LocationResponse, error) {
	if m == "" {
		return nil, nil, errors.New("Empty query string")
	}

	loc, err := p.locationQuery(m)
	if err != nil {
		return nil, nil, err
	}

	link := fmt.Sprintf("https://api.forecast.io/forecast/%s/%.4f,%.4f",
		p.Key,
		loc.Results[0].Geometry.Location.Lat,
		loc.Results[0].Geometry.Location.Lon)

	r, err := http.Get(link)
	if err != nil {
		return nil, nil, err
	}

	f := ForecastResponse{}
	dec := json.NewDecoder(r.Body)
	defer r.Body.Close()
	dec.Decode(&f)

	return &f, loc, nil
}

type ForecastPlugin struct {
	Bot *seabird.Bot
	Key string
}

func init() {
	seabird.RegisterPlugin("forecast", NewForecastPlugin)
}

func NewForecastPlugin(b *seabird.Bot, c json.RawMessage) {
	p := &ForecastPlugin{Bot: b}

	err := json.Unmarshal(c, &p.Key)
	if err != nil {
		fmt.Println(err)
	}

	b.RegisterFunction("fforecast", p.ForecastDaily)
	b.RegisterFunction("fweather", p.ForecastCurrent)
}

func (p *ForecastPlugin) ForecastDaily(e *irc.Event) {
	f, l, err := p.forecastQuery(e.Message())
	if err != nil {
		p.Bot.MentionReply(e, "%s", err.Error())
		return
	}

	p.Bot.MentionReply(e, "7 day forecast for %s.", l.Results[0].Address)
	for _, block := range f.Daily.Data {
		p.Bot.MentionReply(e,
			"High %.2f, Low %.2f, %s",
			block.TemperatureMax,
			block.TemperatureMin,
			block.Summary)
	}
}

func (p *ForecastPlugin) ForecastCurrent(e *irc.Event) {
	f, l, err := p.forecastQuery(e.Message())
	if err != nil {
		p.Bot.MentionReply(e, "%s", err.Error())
		return
	}

	p.Bot.MentionReply(e,
		"%s. Currently %.1f. %s.",
		l.Results[0].Address,
		f.Currently.Temperature,
		f.Currently.Summary)
}
