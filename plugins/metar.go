package plugins

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
)

func init() {
	bot.RegisterPlugin("metar", NewMetarPlugin)
}

func metar(code string) string {
	for _, letter := range code {
		if !unicode.IsDigit(letter) && !unicode.IsLetter(letter) {
			return "Not a valid airport code"
		}
	}

	resp, err := http.Get(fmt.Sprintf("http://weather.noaa.gov/pub/data/observations/metar/stations/%s.TXT", code))
	if err != nil {
		return "NOAA appears to be down"
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "Station does not exist"
	}

	in := bufio.NewReader(resp.Body)
	for {
		line, err := in.ReadString('\n')
		if err != nil {
			break
		}

		if strings.HasPrefix(line, code+" ") {
			return strings.TrimSpace(line)
		}
	}

	return "No results"
}

type MetarPlugin struct{}

func NewMetarPlugin(m *mux.CommandMux) error {
	m.Event("metar", "[station]", Metar)

	return nil
}

func Metar(c *irc.Client, e *irc.Event) {
	if !e.FromChannel() {
		return
	}

	c.MentionReply(e, "%s", metar(e.Trailing()))
}
