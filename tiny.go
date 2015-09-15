package plugins

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/belak/seabird/bot"
	"github.com/belak/irc"
)

func init() {
	bot.RegisterPlugin("tiny", NewTinyPlugin)
}

type ShortenResult struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	LongURL string `json:"longUrl"`
}

type TinyPlugin struct{}

func NewTinyPlugin(b *bot.Bot) (bot.Plugin, error) {
	p := &TinyPlugin{}

	b.CommandMux.Event("tiny", Shorten, &bot.HelpInfo{
		"<url>",
		"Shortens given URL",
	})

	return p, nil
}

func Shorten(b *bot.Bot, m *irc.Message) {
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

		sr := &ShortenResult{}
		err = json.NewDecoder(resp.Body).Decode(sr)
		if err != nil {
			b.MentionReply(m, "Error reading server response")
			return
		}

		b.MentionReply(m, sr.ID)
	}()
}
