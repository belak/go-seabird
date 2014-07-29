package plugins

import (
	seabird ".."
	"../util"

	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/thoj/go-ircevent"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

type LastAddress struct {
	Nick     string
	Location util.Location
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

func (p *ForecastPlugin) forecastQuery(loc util.Coordinates) (*ForecastResponse, error) {

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
	dec.Decode(&f)

	return &f, nil
}

type ForecastPlugin struct {
	Bot *seabird.Bot
	Key string
	C   *mgo.Collection
}

func init() {
	seabird.RegisterPlugin("forecast", NewForecastPlugin)
}

func NewForecastPlugin(b *seabird.Bot, c json.RawMessage) {
	p := &ForecastPlugin{Bot: b, C: b.DB.C("weather")}

	err := json.Unmarshal(c, &p.Key)
	if err != nil {
		fmt.Println(err)
	}

	b.RegisterFunction("forecast", p.ForecastDaily)
	b.RegisterFunction("weather", p.ForecastCurrent)
}

func (p *ForecastPlugin) getLocation(e *irc.Event) (*util.Location, error) {
	l := e.Message()

	loc, err := util.FetchLocation(l)
	if err == nil {
		return loc, nil
	} else if l != "" {
		return nil, err
	}

	la := &LastAddress{}
	cerr := p.C.Find(bson.M{"nick": e.Nick}).One(la)
	if cerr != nil {
		// intentionally use the err from other call.
		// not finding the entry in the DB is ok.
		return nil, err
	}

	return &la.Location, nil
}

func (p *ForecastPlugin) saveLocation(e *irc.Event, loc *util.Location) {
	la := LastAddress{e.Nick, *loc}
	p.C.Upsert(bson.M{"nick": e.Nick}, la)
}

func (p *ForecastPlugin) ForecastDaily(e *irc.Event) {
	loc, err := p.getLocation(e)
	if err != nil {
		p.Bot.MentionReply(e, "%s", err.Error())
		return
	}

	fc, err := p.forecastQuery(loc.Coords)
	if err != nil {
		p.Bot.MentionReply(e, "%s", err.Error())
		return
	}

	p.Bot.MentionReply(e, "3 day forecast for %s.", loc.Address)
	for _, block := range fc.Daily.Data[1:4] {
		day := time.Unix(block.Time, 0).Weekday()

		p.Bot.MentionReply(e,
			"%s: High %.2f, Low %.2f, Humidity %.f%%. %s.",
			day,
			block.TemperatureMax,
			block.TemperatureMin,
			block.Humidity*100,
			block.Summary)
	}

	p.saveLocation(e, loc)
}

func (p *ForecastPlugin) ForecastCurrent(e *irc.Event) {
	loc, err := p.getLocation(e)
	if err != nil {
		p.Bot.MentionReply(e, "%s", err.Error())
		return
	}

	fc, err := p.forecastQuery(loc.Coords)
	if err != nil {
		p.Bot.MentionReply(e, "%s", err.Error())
		return
	}

	today := fc.Daily.Data[0]
	p.Bot.MentionReply(e,
		"%s. Currently %.1f. High %.2f, Low %.2f, Humidity %.f%%. %s",
		loc.Address,
		fc.Currently.Temperature,
		today.TemperatureMax,
		today.TemperatureMin,
		fc.Currently.Humidity*100,
		fc.Currently.Summary)

	p.saveLocation(e, loc)
}
