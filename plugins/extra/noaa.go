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

	"github.com/go-xorm/xorm"

	seabird "github.com/belak/go-seabird"
)

// NOAAStation is a simple cache which will store a user's last-requested
// NOAA station.
type NOAAStation struct {
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
	err := p.db.Sync(NOAAStation{})
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

func (p *noaaPlugin) getStation(b *seabird.Bot, r *seabird.Request) (string, error) {
	l := r.Message.Trailing()

	target := &NOAAStation{Nick: r.Message.Prefix.Name}

	// If it's an empty string, check the cache
	if l == "" {
		found, err := p.db.Get(target)
		if err != nil || !found {
			return "", fmt.Errorf("Could not find a location for %q", r.Message.Prefix.Name)
		}

		return target.Station, nil
	}

	newStation := &NOAAStation{
		Nick:    r.Message.Prefix.Name,
		Station: strings.ToUpper(l),
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

func (p *noaaPlugin) metarCallback(b *seabird.Bot, r *seabird.Request) {
	station, err := p.getStation(b, r)
	if err != nil {
		b.MentionReply(r, "%s", err.Error())
		return
	}

	resp, err := NOAALookup("http://tgftp.nws.noaa.gov/data/observations/metar/stations/%s.TXT", station)
	if err != nil {
		b.MentionReply(r, "Error: %s", err)
		return
	}

	b.MentionReply(r, "%s", resp)
}

func (p *noaaPlugin) tafCallback(b *seabird.Bot, r *seabird.Request) {
	station, err := p.getStation(b, r)
	if err != nil {
		b.MentionReply(r, "%s", err.Error())
		return
	}

	resp, err := NOAALookup("http://tgftp.nws.noaa.gov/data/forecasts/taf/stations/%s.TXT", station)
	if err != nil {
		b.MentionReply(r, "Error: %s", err)
		return
	}

	b.MentionReply(r, "%s", resp)
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
