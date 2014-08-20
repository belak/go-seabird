package seabird

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

	"bitbucket.org/belak/irc"
	"bitbucket.org/belak/seabird/bot"
)

func init() {
	bot.RegisterPlugin("forecast", NewForecastPlugin)
}

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

type ForecastPlugin struct {
	Key           string
	CacheDuration string
	fc            *mgo.Collection
	loc           *mgo.Collection
}

func NewForecastPlugin(b *bot.Bot) (bot.Plugin, error) {
	p := &ForecastPlugin{}
	err := p.Reload(b)
	if err != nil {
		return nil, err
	}

	b.Command("weather", "TODO", p.Weather)
	b.Command("forecast", "TODO", p.Forecast)

	return p, nil
}

func (p *ForecastPlugin) Reload(b *bot.Bot) error {
	err := b.LoadConfig("forecast", p)
	if err != nil {
		return err
	}

	p.loc = b.DB.C("forecast.location")
	p.fc = b.DB.C("forecast.cache")

	// We have to drop this collection in order to update
	// the entry retention policy. mongo doesn't let us change
	// it once we've already set it up.
	p.fc.DropCollection()

	ttl, err := time.ParseDuration(p.CacheDuration)
	if err != nil {
		return err
	}

	p.fc.EnsureIndex(mgo.Index{
		Key:         []string{"created"},
		ExpireAfter: ttl,
	})

	return nil
}

func (p *ForecastPlugin) forecastQuery(loc Coordinates) (*ForecastResponse, error) {

	e := ForecastCacheEntry{}
	err := p.fc.Find(bson.M{"coordinates": loc}).One(&e)
	if err == nil {
		return &e.Response, nil
	}

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

	e = ForecastCacheEntry{
		Coordinates: loc,
		Response:    f,
		Created:     time.Now(),
	}
	p.fc.Insert(e)

	return &f, nil
}

func (p *ForecastPlugin) getLocation(e *irc.Event) (*Location, error) {
	l := strings.TrimSpace(e.Trailing())

	loc, err := FetchLocation(l)
	if err == nil {
		return loc, nil
	} else if l != "" {
		// NOTE: we're checking whether the initial query was empty
		// here, not the result from FetchLocation. If FetchLocation
		// returns an error and location was not provided, we need to
		// check our location cache before we decide what to do next.
		return nil, err
	}

	la := &LastAddress{}
	cerr := p.loc.Find(bson.M{"nick": e.Identity.Nick}).One(la)
	if cerr != nil {
		// intentionally use the err from other call.
		// not finding the entry in the DB is ok.
		return nil, err
	}

	return &la.Location, nil
}

func (p *ForecastPlugin) saveLocation(e *irc.Event, loc *Location) {
	la := LastAddress{e.Identity.Nick, *loc}
	p.loc.Upsert(bson.M{"nick": e.Identity.Nick}, la)
}

func (p *ForecastPlugin) Forecast(b *bot.Bot, e *irc.Event) {
	loc, err := p.getLocation(e)
	if err != nil {
		b.MentionReply(e, "%s", err.Error())
		return
	}

	fc, err := p.forecastQuery(loc.Coords)
	if err != nil {
		b.MentionReply(e, "%s", err.Error())
		return
	}

	b.MentionReply(e, "3 day forecast for %s.", loc.Address)
	for _, block := range fc.Daily.Data[1:4] {
		day := time.Unix(block.Time, 0).Weekday()

		b.MentionReply(e,
			"%s: High %.2f, Low %.2f, Humidity %.f%%. %s",
			day,
			block.TemperatureMax,
			block.TemperatureMin,
			block.Humidity*100,
			block.Summary)
	}

	p.saveLocation(e, loc)
}

func (p *ForecastPlugin) Weather(b *bot.Bot, e *irc.Event) {
	loc, err := p.getLocation(e)
	if err != nil {
		b.MentionReply(e, "%s", err.Error())
		return
	}

	fc, err := p.forecastQuery(loc.Coords)
	if err != nil {
		b.MentionReply(e, "%s", err.Error())
		return
	}

	today := fc.Daily.Data[0]
	b.MentionReply(e,
		"%s. Currently %.1f. High %.2f, Low %.2f, Humidity %.f%%. %s.",
		loc.Address,
		fc.Currently.Temperature,
		today.TemperatureMax,
		today.TemperatureMin,
		fc.Currently.Humidity*100,
		fc.Currently.Summary)

	p.saveLocation(e, loc)
}
