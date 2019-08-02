package extra

import (
	"strings"

	"github.com/lrstanley/girc"
	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/plugins/utils"
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

func newWikiPlugin(c *girc.Client) {
	c.Handlers.AddBg(seabird.PrefixCommand("wiki"), wikiCallback)

	/*
		cm.Event("wiki", wikiCallback, &seabird.HelpInfo{
			Usage:       "<topic>",
			Description: "Retrieves first section from most relevant Wikipedia article to given topic",
		})
	*/
}

func transformQuery(query string) string {
	return strings.Replace(query, " ", "_", -1)
}

func wikiCallback(c *girc.Client, e girc.Event) {
	go func() {
		if e.Last() == "" {
			c.Cmd.ReplyTof(e, "Query required")
			return
		}

		wr := &wikiResponse{}
		err := utils.GetJSON(
			"http://en.wikipedia.org/w/api.php?format=json&action=parse&page="+transformQuery(e.Last()),
			wr)
		if err != nil {
			c.Cmd.ReplyTof(e, "%s", err)
			return
		}

		z, err := html.Parse(strings.NewReader(wr.Parse.Text.Data))
		if err != nil {
			c.Cmd.ReplyTof(e, "%s", err)
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
				c.Cmd.ReplyTof(e, "%s", t)
				return
			}
		}

		c.Cmd.ReplyTof(e, "Error finding text")
	}()
}
