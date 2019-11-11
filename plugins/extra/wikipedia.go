package extra

import (
	"strings"

	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/internal"
)

func init() {
	seabird.RegisterPlugin("wiki", newWikiPlugin)
}

type wikiResponse struct {
	Parse struct {
		Title string `json:"title"`
		Text  struct {
			Data string `json:"*"`
		} `json:"text"`
	} `json:"parse"`
}

func newWikiPlugin(b *seabird.Bot) error {
	cm := b.CommandMux()

	cm.Event("wiki", wikiCallback, &seabird.HelpInfo{
		Usage:       "<topic>",
		Description: "Retrieves first section from most relevant Wikipedia article to given topic",
	})

	return nil
}

func transformQuery(query string) string {
	return strings.Replace(query, " ", "_", -1)
}

func wikiCallback(r *seabird.Request) {
	go func() {
		if r.Message.Trailing() == "" {
			r.MentionReply("Query required")
			return
		}

		wr := &wikiResponse{}
		err := internal.GetJSON(
			"http://en.wikipedia.org/w/api.php?format=json&action=parse&page="+transformQuery(r.Message.Trailing()),
			wr)
		if err != nil {
			r.MentionReply("%s", err)
			return
		}

		z, err := html.Parse(strings.NewReader(wr.Parse.Text.Data))
		if err != nil {
			r.MentionReply("%s", err)
			return
		}

		n, ok := scrape.Find(z, scrape.ByTag(atom.P))
		if ok {
			t := scrape.Text(n)

			if len(t) > 256 {
				t = t[:253]
				t += "..."
			}

			if t != "" {
				r.MentionReply("%s", t)
				return
			}
		}

		r.MentionReply("Error finding text")
	}()
}
