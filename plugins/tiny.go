package plugins

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/belak/irc"
	"github.com/belak/go-seabird/bot"
)

func init() {
	bot.RegisterPlugin("tiny", NewTinyPlugin)
}

type shortenResult struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	LongURL string `json:"longUrl"`
}

func NewTinyPlugin(b *bot.Bot) (bot.Plugin, error) {
	b.CommandMux.Event("tiny", shorten, &bot.HelpInfo{
		Usage:       "<url>",
		Description: "Shortens given URL",
	})

	return nil, nil
}

func shorten(b *bot.Bot, m *irc.Message) {
	go func() {
		if m.Trailing() == "" {
			b.MentionReply(m, "URL required")
			return
		}

		url := "https://www.googleapis.com/urlshortener/v1/url"

		data := map[string]string{"longUrl": m.Trailing()}
		out, err := json.Marshal(data)
		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(out))
		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}
		defer resp.Body.Close()

		sr := &shortenResult{}
		err = json.NewDecoder(resp.Body).Decode(sr)
		if err != nil {
			b.MentionReply(m, "Error reading server response")
			return
		}

		b.MentionReply(m, sr.ID)
	}()
}
