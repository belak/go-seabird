package plugins

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
)

func init() {
	bot.RegisterPlugin("metar", NewMetarPlugin)
}

func NewMetarPlugin(b *bot.Bot) (bot.Plugin, error) {
	b.CommandMux.Event("metar", metarCallback, &bot.HelpInfo{
		Usage:       "<station>",
		Description: "Gives METAR report for given airport code",
	})

	return nil, nil
}

func metarCallback(b *bot.Bot, m *irc.Message) {
	if !m.FromChannel() {
		return
	}

	b.MentionReply(m, "%s", Metar(m.Trailing()))
}

// Metar is a simple function which takes a string representing the
// airport code and returns a string representing the response or an
// error.
func Metar(code string) string {
	code = strings.ToUpper(code)

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
