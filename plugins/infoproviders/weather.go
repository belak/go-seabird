package infoproviders

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
)

type LastAddress struct {
	Nick     string
	Location Location
}

// Basic structures from https://github.com/mlbright/forecast/blob/master/v2/forecast.go
// TODO: cleanup
type DataPoint struct {
	Time                   int64
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

type ForecastCacheEntry struct {
	Coordinates Coordinates
	Created     time.Time
	Response    ForecastResponse
}

type WeatherProvider struct {
	Key string
	// CacheDuration string
}

func NewWeatherProvider(b *bot.Bot, m *mux.CommandMux, mp *MorningPlugin) (*WeatherProvider, error) {
	p := &WeatherProvider{}

	err := b.Config("forecast", p)
	if err != nil {
		return nil, err
	}

	mp.Register("weather", p)

	return p, nil
}

func init() {
	bot.RegisterPlugin("infoprovider:weather", NewWeatherProvider)
}

func (p *WeatherProvider) Get() (*ForecastResponse, error) {
	loc, err := p.GetLocation("94030")
	if err != nil {
		return nil, err
	}

	return p.ForecastQuery(loc.Coords)
}

func (p *WeatherProvider) ForecastQuery(loc Coordinates) (*ForecastResponse, error) {
	link := fmt.Sprintf("https://api.forecast.io/forecast/%s/%.4f,%.4f",
		p.Key,
		loc.Lat,
		loc.Lon)

	r, err := http.Get(link)
	if err != nil {
		return nil, err
	}

	f := ForecastResponse{}
	dec := json.NewDecoder(r.Body)
	defer r.Body.Close()

	err = dec.Decode(&f)
	if err != nil {
		return nil, err
	}

	return &f, nil
}

func (p *WeatherProvider) GetLocation(where string) (*Location, error) {
	loc, err := FetchLocation(where)
	if err != nil {
		return nil, err
	}

	return loc, nil
}
