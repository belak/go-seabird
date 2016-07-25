package plugins

import (
	"net/http"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/Unknwon/com"
	"github.com/belak/go-seabird/bot"
	"github.com/belak/irc"
	"github.com/yhat/scrape"
)

func init() {
	bot.RegisterPlugin("wiki", newWikiPlugin)
}

type wikiResponse struct {
	Parse struct {
		Title string `json:"title"`
		Text  struct {
			Data string `json:"*"`
		} `json:"text"`
	} `json:"parse"`
}

func newWikiPlugin(b *bot.Bot) (bot.Plugin, error) {
	b.CommandMux.Event("wiki", wikiCallback, &bot.HelpInfo{
		Usage:       "<topic>",
		Description: "Retrieves first section from most relevant Wikipedia article to given topic",
	})

	return nil, nil
}

func transformQuery(query string) string {
	return strings.Replace(query, " ", "_", -1)
}

func wikiCallback(b *bot.Bot, m *irc.Message) {
	go func() {
		if m.Trailing() == "" {
			b.MentionReply(m, "Query required")
			return
		}

		wr := &wikiResponse{}
		err := com.HttpGetJSON(
			&http.Client{},
			"http://en.wikipedia.org/w/api.php?format=json&action=parse&page="+transformQuery(m.Trailing()),
			wr,
		)
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
