package plugins

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Unknwon/com"
	"github.com/belak/go-seabird/seabird"
	"github.com/belak/irc"
	"github.com/jmoiron/sqlx"
)

func init() {
	seabird.RegisterPlugin("forecast", newForecastPlugin)
}

// DataPoint represents a point at a specific point in time,
// Basic structures from https://github.com/mlbright/forecast/blob/master/v2/forecast.go
// TODO: cleanup
type dataPoint struct {
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

type dataBlock struct {
	Summary string
	Icon    string
	Data    []dataPoint
}

type alert struct {
	Title   string
	Expires float64
	URI     string
}

type flags struct {
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

type forecastResponse struct {
	Latitude  float64
	Longitude float64
	Timezone  string
	Offset    float64
	Currently dataPoint
	Minutely  dataBlock
	Hourly    dataBlock
	Daily     dataBlock
	Alerts    []alert
	Flags     flags
	APICalls  int
}

type forecastPlugin struct {
	Key string
	db  *sqlx.DB
	// CacheDuration string
}

func newForecastPlugin(b *seabird.Bot, cm *seabird.CommandMux, db *sqlx.DB) error {
	p := &forecastPlugin{db: db}

	err := b.Config("forecast", p)
	if err != nil {
		return err
	}

	cm.Event("weather", p.weatherCallback, &seabird.HelpInfo{
		Usage:       "<location>",
		Description: "Retrieves current weather for given location",
	})

	cm.Event("forecast", p.forecastCallback, &seabird.HelpInfo{
		Usage:       "<location>",
		Description: "Retrieves three-day forecast for given location",
	})

	return nil
}

func (p *forecastPlugin) forecastQuery(loc *Location) (*forecastResponse, error) {
	link := fmt.Sprintf("https://api.forecast.io/forecast/%s/%.4f,%.4f",
		p.Key,
		loc.Lat,
		loc.Lon)

	f := forecastResponse{}
	err := com.HttpGetJSON(&http.Client{}, link, f)
	if err != nil {
		return nil, err
	}

	return &f, nil
}

func (p *forecastPlugin) getLocation(m *irc.Message) (*Location, error) {
	var err error

	l := m.Trailing()
	loc := &Location{}

	// If it's an empty string, check the cache
	if l == "" {
		err = p.db.Get(loc, "SELECT address, lat, lon FROM forecast_location WHERE nick=$1", m.Prefix.Name)
		if err != nil {
			return nil, fmt.Errorf("Could not find a location for %q", m.Prefix.Name)
		}
	} else {
		loc, err = FetchLocation(l)
		if err != nil {
			return nil, err
		}

		_, err = p.db.Exec("INSERT INTO forecast_location (nick, address, lat, lon) VALUES ($1, $2, $3, $4)",
			m.Prefix.Name, loc.Address, loc.Lat, loc.Lon,
		)
		if err == nil {
			return loc, nil
		}

		_, err = p.db.Exec("UPDATE forecast_location SET address=$1, lat=$2, lon=$3 WHERE nick=$4",
			loc.Address, loc.Lat, loc.Lon, m.Prefix.Name,
		)
		if err != nil {
			return nil, err
		}
	}

	return loc, nil
}

func (p *forecastPlugin) forecastCallback(b *seabird.Bot, m *irc.Message) {
	loc, err := p.getLocation(m)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	fc, err := p.forecastQuery(loc)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	b.MentionReply(m, "3 day forecast for %s.", loc.Address)
	for _, block := range fc.Daily.Data[1:4] {
		day := time.Unix(block.Time, 0).Weekday()

		b.MentionReply(m,
			"%s: High %.2f, Low %.2f, Humidity %.f%%. %s",
			day,
			block.TemperatureMax,
			block.TemperatureMin,
			block.Humidity*100,
			block.Summary)
	}
}

func (p *forecastPlugin) weatherCallback(b *seabird.Bot, m *irc.Message) {
	loc, err := p.getLocation(m)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	fc, err := p.forecastQuery(loc)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	today := fc.Daily.Data[0]
	b.MentionReply(m,
		"%s. Currently %.1f. High %.2f, Low %.2f, Humidity %.f%%. %s.",
		loc.Address,
		fc.Currently.Temperature,
		today.TemperatureMax,
		today.TemperatureMin,
		fc.Currently.Humidity*100,
		fc.Currently.Summary)
}
