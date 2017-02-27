package extra

import (
	"fmt"
	"strconv"
	"time"

	"github.com/go-xorm/xorm"
	darksky "github.com/mlbright/darksky/v2"

	"github.com/belak/go-seabird"
	"github.com/belak/nut"
	"github.com/go-irc/irc"
)

func init() {
	seabird.RegisterPlugin("forecast", newForecastPlugin)
}

type forecastPlugin struct {
	Key string
	db  *xorm.Engine
}

// forecastLocation is a simple cache which will store the lat and lon of a
// geocoded location, along with the user who requested this be their home
// location.
type forecastLocation struct {
	Nick    string
	Address string
	Lat     float64
	Lon     float64
}

// ToLocation converts a ForecastLocation to a generic location
func (fl *forecastLocation) ToLocation() *Location {
	return &Location{
		Address: fl.Address,
		Lat:     fl.Lat,
		Lon:     fl.Lon,
	}
}

func newForecastPlugin(b *seabird.Bot, cm *seabird.CommandMux, oldDB *nut.DB, db *xorm.Engine) error {
	p := &forecastPlugin{db: db}

	// Ensure the table is created if it doesn't exist
	err := p.db.Sync(&forecastLocation{})
	if err != nil {
		return err
	}

	// TODO: nutdb migration

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

func (p *forecastPlugin) forecastQuery(loc *Location) (*darksky.Forecast, error) {
	return darksky.Get(
		p.Key,
		strconv.FormatFloat(loc.Lat, 'f', 4, 64),
		strconv.FormatFloat(loc.Lon, 'f', 4, 64),
		"now",
		darksky.US,
		darksky.English,
	)
}

func (p *forecastPlugin) getLocation(m *irc.Message) (*Location, error) {
	l := m.Trailing()

	target := &forecastLocation{Nick: m.Prefix.Name}

	// If it's an empty string, check the cache
	if l == "" {
		found, err := p.db.Get(target)
		if err != nil {
			return nil, err
		}
		if !found {
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

	newLocation := &forecastLocation{
		Nick:    m.Prefix.Name,
		Address: loc.Address,
		Lat:     loc.Lat,
		Lon:     loc.Lon,
	}

	sess := p.db.NewSession()
	err = sess.Begin()
	if err != nil {
		return nil, err
	}
	defer sess.Commit()

	found, err := sess.Get(target)
	if err != nil {
		return nil, err
	}

	if found {
		_, err = sess.Update(newLocation, target)
	} else {
		_, err = sess.Insert(newLocation)
	}

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
