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

	seabird "github.com/belak/go-seabird"
	"github.com/go-xorm/xorm"
	irc "gopkg.in/irc.v3"
)

// NOAAStation is a simple cache which will store a user's last-requested
// NOAA station.
type NoaaStation struct {
	ID      int64
	Nick    string `xorm:"unique"`
	Station string
}

type noaaPlugin struct {
	db *xorm.Engine
}

func init() {
	seabird.RegisterPlugin("noaa", newMetarPlugin)
}

func newMetarPlugin(b *seabird.Bot, cm *seabird.CommandMux, db *xorm.Engine) error {
	p := &noaaPlugin{db: db}

	// Ensure DB tables are up to date
	err := p.db.Sync(NoaaStation{})
	if err != nil {
		return err
	}

	cm.Event("metar", p.metarCallback, &seabird.HelpInfo{
		Usage:       "<station>",
		Description: "Gives METAR report for given station",
	})
	cm.Event("taf", p.tafCallback, &seabird.HelpInfo{
		Usage:       "<station>",
		Description: "Gives TAF report for given station",
	})

	return nil
}

func (p *noaaPlugin) getStation(b *seabird.Bot, m *irc.Message) (string, error) {
	l := m.Trailing()

	target := &NoaaStation{Nick: m.Prefix.Name}

	// If it's an empty string, check the cache
	if l == "" {
		found, err := p.db.Get(target)
		if err != nil || !found {
			return "", fmt.Errorf("Could not find a location for %q", m.Prefix.Name)
		}
		return target.Station, nil
	}

	newStation := &NoaaStation{
		Nick:    m.Prefix.Name,
		Station: m.Trailing(),
	}

	_, err := p.db.Transaction(func(s *xorm.Session) (interface{}, error) {
		found, _ := s.Get(target)
		if !found {
			return s.Insert(newStation)
		}

		return s.ID(target.ID).Update(newStation)
	})

	return newStation.Station, err
}

func (p *noaaPlugin) metarCallback(b *seabird.Bot, m *irc.Message) {
	station, err := p.getStation(b, m)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	r, err := p.NOAALookup("http://tgftp.nws.noaa.gov/data/observations/metar/stations/%s.TXT", station)
	if err != nil {
		b.MentionReply(m, "Error: %s", err)
		return
	}

	b.MentionReply(m, "%s", r)
}

func (p *noaaPlugin) tafCallback(b *seabird.Bot, m *irc.Message) {
	station, err := p.getStation(b, m)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	r, err := p.NOAALookup("http://tgftp.nws.noaa.gov/data/forecasts/taf/stations/%s.TXT", station)
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
func (p *noaaPlugin) NOAALookup(urlFormat, code string) (string, error) {
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
