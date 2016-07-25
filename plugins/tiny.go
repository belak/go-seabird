package plugins

import (
	"net/http"

	"github.com/Unknwon/com"
	"github.com/belak/go-seabird/bot"
	"github.com/belak/irc"
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
		sr := &shortenResult{}
		err := com.HttpPostJSON(&http.Client{}, url, data, sr)
		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}

		b.MentionReply(m, sr.ID)
	}()
}
