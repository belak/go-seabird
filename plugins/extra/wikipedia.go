package extra

import (
	"strings"

	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/plugins/utils"
	irc "gopkg.in/irc.v3"
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

func newWikiPlugin(cm *seabird.CommandMux) {
	cm.Event("wiki", wikiCallback, &seabird.HelpInfo{
		Usage:       "<topic>",
		Description: "Retrieves first section from most relevant Wikipedia article to given topic",
	})
}

func transformQuery(query string) string {
	return strings.Replace(query, " ", "_", -1)
}

func wikiCallback(b *seabird.Bot, m *irc.Message) {
	go func() {
		if m.Trailing() == "" {
			b.MentionReply(m, "Query required")
			return
		}

		wr := &wikiResponse{}
		err := utils.GetJSON(
			"http://en.wikipedia.org/w/api.php?format=json&action=parse&page="+transformQuery(m.Trailing()),
			wr)
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
