package plugins

import (
	"encoding/json"
	"net/http"
	"strings"

	"golang.org/x/net/html"

	"github.com/belak/seabird/bot"
	"github.com/belak/sorcix-irc"
)

type WikiResponse struct {
	Parse struct {
		Title string `json:"title"`
		Text  struct {
			Data string `json:"*"`
		} `json:"text"`
	} `json:"parse"`
}

type WikiPlugin struct{}

func NewWikiPlugin() bot.Plugin {
	return &WikiPlugin{}
}

func (p *WikiPlugin) Register(b *bot.Bot) error {
	b.CommandMux.Event("wiki", Wiki, &bot.HelpInfo{
		"<topic>",
		"Retrieves first section from most relevant Wikipedia article to given topic",
	})

	return nil
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

		// DFS that searches the tree for any node named p then
		// returns the data of that node's first child
		var f func(*html.Node) (string, bool)
		f = func(n *html.Node) (string, bool) {
			// If it's an element and it's a title node, look for a child
			if n.Type == html.ElementNode && n.Data == "p" {
				if n.FirstChild != nil {
					t := ""
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						if c.Type == html.ElementNode && c.FirstChild != nil && c.FirstChild.Type == html.ElementNode {
							// For those pesky <span><spans>s
							continue
						} else if c.Type == html.ElementNode && c.FirstChild != nil {
							t += c.FirstChild.Data
						} else {
							t += c.Data
						}
					}

					// TODO: Remove arbitrary limit
					if len(t) > 256 {
						t = t[:256]
					}

					if t != "" {
						return t, true
					} else {
						return "", false
					}
				}
			}

			// Loop through all nodes and try recursing
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if str, ok := f(c); ok {
					return str, true
				}
			}

			return "", false
		}

		if str, ok := f(z); ok {
			b.MentionReply(m, "%s", str)
		} else {
			b.MentionReply(m, "Error finding text")
		}
	}()
}
