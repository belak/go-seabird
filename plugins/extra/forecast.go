package extra

import (
	"fmt"
	"strconv"
	"time"

	forecast "github.com/mlbright/forecast/v2"

	"github.com/belak/go-seabird"
	"github.com/go-irc/irc"
	"github.com/belak/nut"
)

func init() {
	seabird.RegisterPlugin("forecast", newForecastPlugin)
}

type forecastPlugin struct {
	Key string
	db  *nut.DB
	// CacheDuration string
}

// ForecastLocation is a simple cache which will store the lat and lon of a
// geocoded location, along with the user who requested this be their home
// location.
type ForecastLocation struct {
	Nick    string
	Address string
	Lat     float64
	Lon     float64
}

// ToLocation converts a ForecastLocation to a generic location
func (fl *ForecastLocation) ToLocation() *Location {
	return &Location{
		Address: fl.Address,
		Lat:     fl.Lat,
		Lon:     fl.Lon,
	}
}

func newForecastPlugin(b *seabird.Bot, cm *seabird.CommandMux, db *nut.DB) error {
	p := &forecastPlugin{db: db}

	// Ensure the table is created if it doesn't exist
	err := p.db.EnsureBucket("forecast_location")
	if err != nil {
		return err
	}

	err = b.Config("forecast", p)
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

func (p *forecastPlugin) forecastQuery(loc *Location) (*forecast.Forecast, error) {
	return forecast.Get(
		p.Key,
		strconv.FormatFloat(loc.Lat, 'f', 4, 64),
		strconv.FormatFloat(loc.Lon, 'f', 4, 64),
		"now", forecast.US)
}

func (p *forecastPlugin) getLocation(m *irc.Message) (*Location, error) {
	l := m.Trailing()

	target := &ForecastLocation{Nick: m.Prefix.Name}

	// If it's an empty string, check the cache
	if l == "" {
		err := p.db.View(func(tx *nut.Tx) error {
			bucket := tx.Bucket("forecast_location")
			return bucket.Get(target.Nick, target)
		})
		if err != nil {
			return nil, fmt.Errorf("Could not find a location for %q", m.Prefix.Name)
		}
		return target.ToLocation(), nil
	}

	// If it's not an empty string, we have to look up the location and store
	// it.
	loc, err := FetchLocation(l)
	if err != nil {
		return nil, err
	}

	newLocation := &ForecastLocation{
		Nick:    m.Prefix.Name,
		Address: loc.Address,
		Lat:     loc.Lat,
		Lon:     loc.Lon,
	}

	err = p.db.Update(func(tx *nut.Tx) error {
		bucket := tx.Bucket("forecast_location")
		return bucket.Put(newLocation.Nick, newLocation)
	})

	if err != nil {
		return nil, err
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
		day := time.Unix(int64(block.Time), 0).Weekday()

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
