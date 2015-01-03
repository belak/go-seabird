package plugins

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
)

type ShortenResult struct {
	Kind    string `json:"kind"`
	Id      string `json:"id"`
	LongUrl string `json:"longUrl"`
}

func init() {
	bot.RegisterPlugin("tiny", NewTinyPlugin)
}

func NewTinyPlugin(m *mux.CommandMux) error {
	m.Event("tiny", Shorten, &mux.HelpInfo{
		"<url>",
		"Shortens given URL",
	})

	return nil
}

func Shorten(c *irc.Client, e *irc.Event) {
	go func() {
		if e.Trailing() == "" {
			c.MentionReply(e, "URL required")
			return
		}

		url := "https://www.googleapis.com/urlshortener/v1/url"

		data := map[string]string{"longUrl": e.Trailing()}
		out, err := json.Marshal(data)
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(out))
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}
		defer resp.Body.Close()

		sr := &ShortenResult{}
		err = json.NewDecoder(resp.Body).Decode(sr)
		if err != nil {
			c.MentionReply(e, "Error reading server response")
		}

		c.MentionReply(e, sr.Id)
	}()
}
