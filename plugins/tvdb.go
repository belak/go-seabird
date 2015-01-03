package plugins

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"html"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
)

type TVDBPlugin struct {
	Key string
}

type Series struct {
	XMLName    xml.Name `xml:"Series"`
	Id         string   `xml:"seriesid"`
	Name       string   `xml:"SeriesName"`
	Network    string   `xml:"Network"`
	FirstAired string   `xml:"FirstAired"`
}

type TVDBResponse struct {
	XMLName xml.Name `xml:"Data"`
	Series  []Series `xml:"Series"`
}

type TVDBZipResponse struct {
	XMLName xml.Name `xml:"Data"`
	Series  struct {
		XMLName    xml.Name `xml:"Series"`
		Name       string   `xml:"SeriesName"`
		Network    string   `xml:"Network"`
		FirstAired string   `xml:"FirstAired"`
		Actors     string   `xml:"Actors"`
		Genre      string   `xml:"Genre"`
		Rating     string   `xml:"Rating"`
	} `xml:"Series"`
}

func init() {
	bot.RegisterPlugin("tvdb", NewTVDBPlugin)
}

func NewTVDBPlugin(b *bot.Bot, m *mux.CommandMux) error {
	p := &TVDBPlugin{}

	b.Config("tvdb", p)

	m.Event("tvdb", p.Search, &mux.HelpInfo{
		"<series>",
		"Gives info on TVDB series, including TVDB ID",
	})
	m.Event("series", p.Series, &mux.HelpInfo{
		"<series_id>",
		"Gives expanded info on TVDB series using TVDB ID",
	})

	return nil
}

func (p *TVDBPlugin) Search(c *irc.Client, e *irc.Event) {
	go func() {
		if e.Trailing() == "" {
			c.MentionReply(e, "Series required")
			return
		}

		resp, err := http.Get("http://thetvdb.com/api/GetSeries.php?seriesname=" + url.QueryEscape(e.Trailing()))
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}
		defer resp.Body.Close()

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}

		xmlData := string(data)

		xmlData = html.EscapeString(xmlData)
		xmlData = strings.Replace(xmlData, "&lt;", "<", -1)
		xmlData = strings.Replace(xmlData, "&gt;", ">", -1)

		tr := &TVDBResponse{}
		err = xml.NewDecoder(strings.NewReader(xmlData)).Decode(tr)
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}

		if len(tr.Series) == 0 {
			c.MentionReply(e, "No series found")
			return
		}

		series := tr.Series[0]
		out := series.Name
		if series.Network != "" {
			out += " (" + series.Network + ")"
		}
		if series.FirstAired != "" {
			out += " - " + series.FirstAired
		}
		out += " [id: " + series.Id + "]"

		c.MentionReply(e, "%s", out)
	}()
}

func (p *TVDBPlugin) Series(c *irc.Client, e *irc.Event) {
	go func() {
		if e.Trailing() == "" {
			c.MentionReply(e, "Series required")
			return
		}

		id := e.Trailing()
		language := "en"

		resp, err := http.Get("http://thetvdb.com/api/" + p.Key + "/series/" + id + "/all/" + language + ".zip")
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}

		// Create zipfile from stream
		zipfile, err := zip.NewReader(bytes.NewReader([]byte(body)), int64(len([]byte(body))))
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}

		// Find the proper xml file
		for _, f := range zipfile.File {
			if f.Name == language+".xml" {
				zipped, err := f.Open()
				if err != nil {
					c.MentionReply(e, "%s", err)
					return
				}
				defer zipped.Close()

				body, err := ioutil.ReadAll(zipped)
				if err != nil {
					c.MentionReply(e, "%s", err)
					return
				}

				data := string(body)

				data = html.EscapeString(data)
				data = strings.Replace(data, "&lt;", "<", -1)
				data = strings.Replace(data, "&gt;", ">", -1)

				v := TVDBZipResponse{}
				err = xml.Unmarshal([]byte(data), &v)
				if err != nil {
					c.MentionReply(e, "%s", err)
					return
				}

				s := v.Series
				out := s.Name + "."
				if s.Rating != "" {
					out += " Rated " + s.Rating + "/10."
				}
				if s.FirstAired != "" && s.Network != "" {
					out += " First aired " + s.FirstAired + " on " + s.Network + "."
				}
				if s.Actors != "" {
					out += " Actors: " + changeBars(s.Actors) + "."
				}
				if s.Genre != "" {
					out += " Genre(s): " + changeBars(s.Genre) + "."
				}
				c.MentionReply(e, "%s", out)
			}
		}
	}()
}

func changeBars(in string) string {
	if in == "" {
		return ""
	}

	if in[0] == '|' {
		in = in[1 : len(in)-1]
	}

	return strings.Replace(in, "|", ", ", -1)
}
