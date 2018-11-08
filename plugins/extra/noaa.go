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
	irc "github.com/go-irc/irc"
)

func init() {
	seabird.RegisterPlugin("noaa", newMetarPlugin)
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
	r, err := NOAALookup("http://tgftp.nws.noaa.gov/data/observations/metar/stations/%s.TXT", m.Trailing())
	if err != nil {
		b.MentionReply(m, "Error: %s", err)
		return
	}

	b.MentionReply(m, "%s", r)
}

func tafCallback(b *seabird.Bot, m *irc.Message) {
	r, err := NOAALookup("http://tgftp.nws.noaa.gov/data/forecasts/taf/stations/%s.TXT", m.Trailing())
	if err != nil {
		b.MentionReply(m, "Error: %s", err)
		return
	}

	b.MentionReply(m, "%s", r)
}

// NOAALookup takes the given formatted url and an airport code and tries to
// look up the raw data. The first line is skipped, as that is generally the
// date and the rest of the lines are joined together with a maximum of one
// space between them.
func NOAALookup(urlFormat, code string) (string, error) {
	code = strings.ToUpper(code)

	for _, letter := range code {
		if !unicode.IsDigit(letter) && !unicode.IsLetter(letter) {
			return "", errors.New("Not a valid airport code")
		}
	}

	resp, err := http.Get(fmt.Sprintf(urlFormat, code))
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

		// We skip the first line as it contains the date.
		if !first {
			first = true
			continue
		}

		out.WriteString(" " + strings.TrimSpace(line))
	}

	return strings.TrimSpace(out.String()), nil
}
