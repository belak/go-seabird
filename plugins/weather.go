package plugins

import (
	"../../seabird"
	"encoding/json"
	"fmt"
	"github.com/thoj/go-ircevent"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var country_replacements = map[string]string{
	"US": "USA",
	"United States of America": "USA",
}

type WeatherResponse struct {
	Msg  string `json:"message"`
	List []struct {
		Name string `json:"name"`
		Main struct {
			Temp    float32 `json:"temp"`
			TempMin float32 `json:"temp_min"`
			TempMax float32 `json:"temp_max"`
		} `json:"main"`
		Sys struct {
			Country string `json:"country"`
		} `json:"sys"`
		Weather []struct {
			Description string `json:"description"`
		} `json:"weather"`
	} `json:"list"`
}

func init() {
	seabird.RegisterPlugin("weather", NewWeatherPlugin)
}

type WeatherPlugin struct {
	Bot *seabird.Bot
}

func NewWeatherPlugin(b *seabird.Bot, c json.RawMessage) {
	p := &WeatherPlugin{b}
	b.RegisterFunction("weather", p.Weather)
}

func (p *WeatherPlugin) Weather(e *irc.Event) {
	var query string = strings.TrimSpace(e.Message)
	if _, err := strconv.Atoi(query); err == nil {
		// It's a number - append ,USA
		query = query + ",USA"
	}
	resp, err := http.Get(fmt.Sprintf("http://api.openweathermap.org/data/2.5/find?mode=json&units=imperial&q=%s", url.QueryEscape(query)))
	if err != nil {
		p.Bot.MentionReply(e, "an error occured")
		return
	}

	weather := WeatherResponse{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&weather)
	if err != nil || len(weather.List) < 1 || len(weather.List[0].Weather) < 1 {
		p.Bot.MentionReply(e, "invalid api response")
		return
	}

	// TODO: Improve this
	loc := weather.List[0]
	if replacement, ok := country_replacements[loc.Sys.Country]; ok {
		loc.Sys.Country = replacement
	}

	p.Bot.MentionReply(e,
		"%s, %s. Currently %.1f. High %.2f, Low %.2f, %s",
		loc.Name, loc.Sys.Country,
		loc.Main.Temp, loc.Main.TempMax, loc.Main.TempMin,
		loc.Weather[0].Description)
}
