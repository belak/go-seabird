package extra

import (
	"fmt"

	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/internal"
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

func newTinyPlugin(b *seabird.Bot) error {
	p := &tinyPlugin{}

	if err := b.Config("tiny", p); err != nil {
		return err
	}

	cm := b.CommandMux()

	cm.Event("tiny", p.Shorten, &seabird.HelpInfo{
		Usage:       "<url>",
		Description: "Shortens given URL",
	})

	return nil
}

func (t *tinyPlugin) Shorten(r *seabird.Request) {
	go func() {
		if r.Message.Trailing() == "" {
			r.MentionReply("URL required")
			return
		}

		url := fmt.Sprintf("https://www.googleapis.com/urlshortener/v1/url?key=%s", t.Key)

		data := map[string]string{"longUrl": r.Message.Trailing()}
		sr := &shortenResult{}
		err := internal.PostJSON(url, data, sr)
		if err != nil {
			r.MentionReply("%s", err)
			return
		}

		r.MentionReply(sr.ID)
	}()
}
