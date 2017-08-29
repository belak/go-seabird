package extra

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode"

	"github.com/belak/go-seabird"
	"github.com/go-irc/irc"
)

func init() {
	seabird.RegisterPlugin("metar", newMetarPlugin)
}

func newMetarPlugin(cm *seabird.CommandMux) {
	cm.Event("metar", metarCallback, &seabird.HelpInfo{
		Usage:       "<station>",
		Description: "Gives METAR report for given station",
	})
	cm.Event("taf", tafCallback, &seabird.HelpInfo{
		Usage:       "<station>",
		Description: "Gives TAF report for given station",
	})
}

func metarCallback(b *seabird.Bot, m *irc.Message) {
	if !m.FromChannel() {
		return
	}

	r, err := Metar(m.Trailing())
	if err != nil {
		b.MentionReply(m, "Error: %s", err)
		return
	}

	b.MentionReply(m, "%s", r)
}

// Metar is a simple function which takes a string representing the
// airport code and returns a string representing the response or an
// error.
func Metar(code string) (string, error) {
	code = strings.ToUpper(code)

	for _, letter := range code {
		if !unicode.IsDigit(letter) && !unicode.IsLetter(letter) {
			return "", errors.New("Not a valid airport code")
		}
	}

	resp, err := http.Get(fmt.Sprintf("http://tgftp.nws.noaa.gov/data/observations/metar/stations/%s.TXT", code))
	if err != nil {
		return "", errors.New("NOAA appears to be down")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New("Station does not exist")
	}

	in := bufio.NewReader(resp.Body)
	for {
		line, err := in.ReadString('\n')
		if err != nil {
			break
		}

		if strings.HasPrefix(line, code+" ") {
			return strings.TrimSpace(line), nil
		}
	}

	return "", errors.New("No results")
}

func tafCallback(b *seabird.Bot, m *irc.Message) {
	if !m.FromChannel() {
		return
	}

	r, err := TAF(m.Trailing())
	if err != nil {
		b.MentionReply(m, "Error: %s", err)
		return
	}

	b.MentionReply(m, "%s", r)
}

// TAF is a simple function which takes a string representing the
// airport code and returns a string representing the response or an
// error.
func TAF(code string) (string, error) {
	code = strings.ToUpper(code)

	for _, letter := range code {
		if !unicode.IsDigit(letter) && !unicode.IsLetter(letter) {
			return "", errors.New("Not a valid airport code")
		}
	}

	resp, err := http.Get(fmt.Sprintf("http://tgftp.nws.noaa.gov/data/forecasts/taf/stations/%s.TXT", code))
	if err != nil {
		return "", errors.New("NOAA appears to be down")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New("Station does not exist")
	}

	out := &bytes.Buffer{}
	in := bufio.NewReader(resp.Body)
	first := false
	for {
		line, err := in.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return "", errors.New("No results")
		}

		if !first {
			first = true
			continue
		}

		out.WriteString(" " + strings.TrimSpace(line))
	}

	return strings.TrimSpace(out.String()), nil
}
