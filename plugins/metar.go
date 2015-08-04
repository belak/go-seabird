package plugins

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/belak/seabird/bot"
	"github.com/belak/sorcix-irc"
)

func init() {
	bot.RegisterPlugin("metar", NewMetarPlugin)
}

type MetarPlugin struct{}

func NewMetarPlugin(b *bot.Bot) (bot.Plugin, error) {
	p := &MetarPlugin{}

	b.CommandMux.Event("metar", Metar, &bot.HelpInfo{
		"<station>",
		"Gives METAR report for given airport code",
	})

	return p, nil
}

func Metar(b *bot.Bot, m *irc.Message) {
	if !bot.MessageFromChannel(m) {
		return
	}

	b.MentionReply(m, "%s", metar(m.Trailing()))
}

func metar(code string) string {
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
