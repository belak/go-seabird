package extra

import (
	"fmt"
	"net/http"

	"github.com/Unknwon/com"

	"github.com/belak/go-seabird"
	irc "github.com/go-irc/irc/v2"
)

func init() {
	seabird.RegisterPlugin("tiny", newTinyPlugin)
}

type tinyPlugin struct {
	Key string
}

type shortenResult struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	LongURL string `json:"longUrl"`
}

func newTinyPlugin(b *seabird.Bot, cm *seabird.CommandMux) error {
	p := &tinyPlugin{}
	err := b.Config("tiny", p)
	if err != nil {
		return err
	}

	cm.Event("tiny", p.Shorten, &seabird.HelpInfo{
		Usage:       "<url>",
		Description: "Shortens given URL",
	})

	return nil
}

func (t *tinyPlugin) Shorten(b *seabird.Bot, m *irc.Message) {
	go func() {
		if m.Trailing() == "" {
			b.MentionReply(m, "URL required")
			return
		}

		url := fmt.Sprintf("https://www.googleapis.com/urlshortener/v1/url?key=%s", t.Key)

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
