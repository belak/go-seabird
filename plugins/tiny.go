package plugins

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/belak/seabird/bot"
	"github.com/belak/sorcix-irc"
)

type ShortenResult struct {
	Kind    string `json:"kind"`
	Id      string `json:"id"`
	LongUrl string `json:"longUrl"`
}

type TinyPlugin struct{}

func NewTinyPlugin() bot.Plugin {
	return &TinyPlugin{}
}

func (p *TinyPlugin) Register(b *bot.Bot) error {
	b.CommandMux.Event("tiny", Shorten, &bot.HelpInfo{
		"<url>",
		"Shortens given URL",
	})

	return nil
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

		b.MentionReply(m, sr.Id)
	}()
}
