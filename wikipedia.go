package plugins

import (
	"encoding/json"
	"net/http"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/yhat/scrape"
)

func init() {
	bot.RegisterPlugin("wiki", NewWikiPlugin)
}

type WikiResponse struct {
	Parse struct {
		Title string `json:"title"`
		Text  struct {
			Data string `json:"*"`
		} `json:"text"`
	} `json:"parse"`
}

type WikiPlugin struct{}

func NewWikiPlugin(b *bot.Bot) (bot.Plugin, error) {
	p := &WikiPlugin{}

	b.CommandMux.Event("wiki", Wiki, &bot.HelpInfo{
		Usage:       "<topic>",
		Description: "Retrieves first section from most relevant Wikipedia article to given topic",
	})

	return p, nil
}

func transformQuery(query string) string {
	return strings.Replace(query, " ", "_", -1)
}

func Wiki(b *bot.Bot, m *irc.Message) {
	go func() {
		if m.Trailing() == "" {
			b.MentionReply(m, "Query required")
			return
		}

		resp, err := http.Get("http://en.wikipedia.org/w/api.php?format=json&action=parse&page=" + transformQuery(m.Trailing()))
		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}
		defer resp.Body.Close()

		wr := &WikiResponse{}
		err = json.NewDecoder(resp.Body).Decode(wr)
		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}

		z, err := html.Parse(strings.NewReader(wr.Parse.Text.Data))
		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}

		n, ok := scrape.Find(z, scrape.ByTag(atom.P))
		if ok {
			t := scrape.Text(n)

			if len(t) > 256 {
				t = t[:253]
				t = t + "..."
			}

			if t != "" {
				b.MentionReply(m, "%s", t)
				return
			}
		}

		b.MentionReply(m, "Error finding text")
	}()
}
