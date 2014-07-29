package plugins

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"

	"code.google.com/p/go.net/html"
	irc "github.com/thoj/go-ircevent"

	seabird ".."
)

func init() {
	seabird.RegisterPlugin("url", NewURLPlugin)
}

type URLPlugin struct {
	Bot *seabird.Bot
}

// NOTE: This isn't perfect in any sense of the word, but it's pretty close
// and I don't know if it's worth the time to make it better.
var urlRegex = regexp.MustCompile(`https?://[^ ]+`)

func NewURLPlugin(b *seabird.Bot, c json.RawMessage) {
	p := &URLPlugin{b}
	b.RegisterCallback("PRIVMSG", p.Msg)
}

func (p *URLPlugin) Msg(e *irc.Event) {
	for _, url := range urlRegex.FindAllString(e.Message(), -1) {
		go func() {
			r, err := http.Get(url)
			if err != nil {
				continue
			}
			defer r.Body.Close()

			if r.StatusCode != 200 {
				continue
			}

			// We search the first 1K and if a title isn't in there, we deal with it
			z, err := html.Parse(io.LimitReader(r.Body, 1024*1024))
			if err != nil {
				continue
			}

			// DFS that searches the tree for any node named title then
			// returns the data of that node's first child
			var f func(*html.Node) (string, bool)
			f = func(n *html.Node) (string, bool) {
				// If it's an element and it's a title node, look for a child
				if n.Type == html.ElementNode && n.Data == "title" {
					if n.FirstChild != nil {
						return n.FirstChild.Data, true
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
				p.Bot.Reply(e, "Title: %s", str)
			}
		}()
	}
}
