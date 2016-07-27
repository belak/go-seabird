package plugins

import (
	"net/http"

	"github.com/Unknwon/com"

	"github.com/belak/go-seabird/seabird"
	"github.com/belak/irc"
)

func init() {
	seabird.RegisterPlugin("tiny", newTinyPlugin)
}

type shortenResult struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	LongURL string `json:"longUrl"`
}

func newTinyPlugin(cm *seabird.CommandMux) {
	cm.Event("tiny", shorten, &seabird.HelpInfo{
		Usage:       "<url>",
		Description: "Shortens given URL",
	})
}

func shorten(b *seabird.Bot, m *irc.Message) {
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
