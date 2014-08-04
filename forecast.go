package seabird

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bitbucket.org/belak/irc"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
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

type ForecastHandler struct {
	key string
	c   *mgo.Collection
}

func (h *ForecastHandler) forecastQuery(loc Coordinates) (*ForecastResponse, error) {
	link := fmt.Sprintf("https://api.forecast.io/forecast/%s/%.4f,%.4f",
		h.key,
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

func NewForecastHandler(key string, c *mgo.Collection) *ForecastHandler {
	return &ForecastHandler{
		key,
		c,
	}
}

func (h *ForecastHandler) HandleEvent(c *irc.Client, e *irc.Event) {
	if e.Command == "forecast" {
		h.ForecastDaily(c, e)
	} else if e.Command == "weather" {
		h.ForecastCurrent(c, e)
	}
}

func (h *ForecastHandler) getLocation(e *irc.Event) (*Location, error) {
	l := strings.TrimSpace(e.Trailing())

	loc, err := FetchLocation(l)
	if err == nil {
		return loc, nil
	} else if l != "" {
		return nil, err
	}

	la := &LastAddress{}
	cerr := h.c.Find(bson.M{"nick": e.Identity.Nick}).One(la)
	if cerr != nil {
		// intentionally use the err from other call.
		// not finding the entry in the DB is ok.
		return nil, err
	}

	return &la.Location, nil
}

func (h *ForecastHandler) saveLocation(e *irc.Event, loc *Location) {
	la := LastAddress{e.Identity.Nick, *loc}
	h.c.Upsert(bson.M{"nick": e.Identity.Nick}, la)
}

func (h *ForecastHandler) ForecastDaily(c *irc.Client, e *irc.Event) {
	loc, err := h.getLocation(e)
	if err != nil {
		c.MentionReply(e, "%s", err.Error())
		return
	}

	fc, err := h.forecastQuery(loc.Coords)
	if err != nil {
		c.MentionReply(e, "%s", err.Error())
		return
	}

	c.MentionReply(e, "3 day forecast for %s.", loc.Address)
	for _, block := range fc.Daily.Data[1:4] {
		day := time.Unix(block.Time, 0).Weekday()

		c.MentionReply(e,
			"%s: High %.2f, Low %.2f, Humidity %.f%%. %s",
			day,
			block.TemperatureMax,
			block.TemperatureMin,
			block.Humidity*100,
			block.Summary)
	}

	h.saveLocation(e, loc)
}

func (h *ForecastHandler) ForecastCurrent(c *irc.Client, e *irc.Event) {
	loc, err := h.getLocation(e)
	if err != nil {
		c.MentionReply(e, "%s", err.Error())
		return
	}

	fc, err := h.forecastQuery(loc.Coords)
	if err != nil {
		c.MentionReply(e, "%s", err.Error())
		return
	}

	today := fc.Daily.Data[0]
	c.MentionReply(e,
		"%s. Currently %.1f. High %.2f, Low %.2f, Humidity %.f%%. %s.",
		loc.Address,
		fc.Currently.Temperature,
		today.TemperatureMax,
		today.TemperatureMin,
		fc.Currently.Humidity*100,
		fc.Currently.Summary)

	h.saveLocation(e, loc)
}
