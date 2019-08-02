package extra

import (
	"fmt"

	"github.com/lrstanley/girc"

	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/plugins/utils"
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

func newTinyPlugin(b *seabird.Bot, c *girc.Client) error {
	p := &tinyPlugin{}
	err := b.Config("tiny", p)
	if err != nil {
		return err
	}

	c.Handlers.AddBg(seabird.PrefixCommand("tiny"), p.Shorten)

	/*
		cm.Event("tiny", p.Shorten, &seabird.HelpInfo{
			Usage:       "<url>",
			Description: "Shortens given URL",
		})
	*/

	return nil
}

func (t *tinyPlugin) Shorten(c *girc.Client, e girc.Event) {
	go func() {
		if e.Last() == "" {
			c.Cmd.ReplyTof(e, "URL required")
			return
		}

		url := fmt.Sprintf("https://www.googleapis.com/urlshortener/v1/url?key=%s", t.Key)

		data := map[string]string{"longUrl": e.Last()}
		sr := &shortenResult{}
		err := utils.PostJSON(url, data, sr)
		if err != nil {
			c.Cmd.ReplyTof(e, "%s", err)
			return
		}

		c.Cmd.ReplyTof(e, sr.ID)
	}()
}
