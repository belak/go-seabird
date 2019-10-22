package extra

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-xorm/xorm"
	darksky "github.com/mlbright/darksky/v2"
	"googlemaps.github.io/maps"

	seabird "github.com/belak/go-seabird"
	irc "gopkg.in/irc.v3"
)

func init() {
	seabird.RegisterPlugin("forecast", newForecastPlugin)
}

const defaultUnitString = "°"

var unitStrings = map[darksky.Units]string{
	darksky.CA: "°C",
	darksky.SI: "°C",
	darksky.US: "°F",
	darksky.UK: "°C",

	// Documented by darksky, but not the library so we need to hack it in.
	darksky.Units("uk2"): "°C",
}

func getUnit(unit darksky.Units) string {
	if v, ok := unitStrings[unit]; ok {
		return v
	}

	return defaultUnitString
}

type forecastPlugin struct {
	Key        string
	MapsKey    string
	db         *xorm.Engine
	mapsClient *maps.Client
	// CacheDuration string
}

// ForecastLocation is a simple cache which will store the lat and lon of a
// geocoded location, along with the user who requested this be their home
// location.
type ForecastLocation struct {
	ID      int64
	Nick    string `xorm:"unique"`
	Address string
	Lat     float64
	Lon     float64
}

func newForecastPlugin(b *seabird.Bot, cm *seabird.CommandMux, db *xorm.Engine) error {
	p := &forecastPlugin{db: db}

	// Ensure DB tables are up to date
	err := p.db.Sync(ForecastLocation{})
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

	options := []maps.ClientOption{}
	if p.MapsKey != "" {
		options = append(options, maps.WithAPIKey(p.MapsKey))
	}

	p.mapsClient, err = maps.NewClient(options...)
	if err != nil {
		return err
	}

	return nil
}

func (p *forecastPlugin) forecastQuery(loc *ForecastLocation) (*darksky.Forecast, error) {
	return darksky.Get(
		p.Key,
		strconv.FormatFloat(loc.Lat, 'f', 4, 64),
		strconv.FormatFloat(loc.Lon, 'f', 4, 64),
		"now",
		darksky.AUTO,
		darksky.English)
}

func (p *forecastPlugin) getLocation(m *irc.Message) (*ForecastLocation, error) {
	l := m.Trailing()

	target := &ForecastLocation{Nick: m.Prefix.Name}

	// If it's an empty string, check the cache
	if l == "" {
		found, err := p.db.Get(target)
		if err != nil || !found {
			return nil, fmt.Errorf("Could not find a location for %q", m.Prefix.Name)
		}

		return target, nil
	}

	// If it's not an empty string, we have to look up the location and store
	// it.
	res, err := p.mapsClient.Geocode(context.TODO(), &maps.GeocodingRequest{
		Address: l,
	})
	//nolint:gocritic
	if err != nil {
		return nil, err
	} else if len(res) == 0 {
		return nil, errors.New("No location results found")
	} else if len(res) > 1 {
		return nil, errors.New("More than 1 result")
	}

	newLocation := &ForecastLocation{
		Nick:    m.Prefix.Name,
		Address: res[0].FormattedAddress,
		Lat:     res[0].Geometry.Location.Lat,
		Lon:     res[0].Geometry.Location.Lng,
	}

	_, err = p.db.Transaction(func(s *xorm.Session) (interface{}, error) {
		found, _ := s.Get(target)
		if !found {
			return s.Insert(newLocation)
		}

		return s.ID(target.ID).Update(newLocation)
	})

	return newLocation, err
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

	unit := getUnit(darksky.Units(fc.Flags.Units))

	b.MentionReply(m, "3 day forecast for %s.", loc.Address)

	for _, block := range fc.Daily.Data[1:4] {
		day := time.Unix(block.Time, 0).Weekday()

		b.MentionReply(m,
			"%s: High %.2f%s, Low %.2f%s, Humidity %.f%%. %s",
			day,
			block.TemperatureMax,
			unit,
			block.TemperatureMin,
			unit,
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

	unit := getUnit(darksky.Units(fc.Flags.Units))

	today := fc.Daily.Data[0]
	b.MentionReply(m,
		"%s. Currently %.1f%s. High %.2f%s, Low %.2f%s, Humidity %.f%%. %s.",
		loc.Address,
		fc.Currently.Temperature,
		unit,
		today.TemperatureMax,
		unit,
		today.TemperatureMin,
		unit,
		fc.Currently.Humidity*100,
		fc.Currently.Summary)
}
