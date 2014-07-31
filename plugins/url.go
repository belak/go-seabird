package plugins

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"crypto/tls"
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
var titleRegex = regexp.MustCompile(`(?:\s*[\r\n]+\s*)+`)

func NewURLPlugin(b *seabird.Bot, c json.RawMessage) {
	p := &URLPlugin{b}
	b.RegisterCallback("PRIVMSG", p.Msg)
	b.RegisterFunction("down", p.IsItDown)
}

// NOTE: This nasty work is done so we ignore invalid ssl certs
var client = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
	Timeout: 5 * time.Second,
}

func (p *URLPlugin) Msg(e *irc.Event) {
	for _, url := range urlRegex.FindAllString(e.Message(), -1) {
		go func(url string) {
			r, err := client.Get(url)
			if err != nil {
				return
			}
			defer r.Body.Close()

			if r.StatusCode != 200 {
				return
			}

			// We search the first 1K and if a title isn't in there, we deal with it
			z, err := html.Parse(io.LimitReader(r.Body, 1024*1024))
			if err != nil {
				return
			}

			// DFS that searches the tree for any node named title then
			// returns the data of that node's first child
			var f func(*html.Node) (string, bool)
			f = func(n *html.Node) (string, bool) {
				// If it's an element and it's a title node, look for a child
				if n.Type == html.ElementNode && n.Data == "title" {
					if n.FirstChild != nil {
						t := n.FirstChild.Data
						t = titleRegex.ReplaceAllString(t, " ")
						t = strings.TrimSpace(t)

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
				p.Bot.Reply(e, "Title: %s", str)
			}
		}(url)
	}
}

func (p *URLPlugin) IsItDown(e *irc.Event) {
	go func() {
		url, err := url.Parse(strings.TrimSpace(e.Message()))
		if err != nil {
			p.Bot.Reply(e, "URL doesn't appear to be valid")
			return
		}

		if url.Scheme == "" {
			url.Scheme = "http"
		}

		r, err := client.Head(url.String())
		if err != nil || r.StatusCode != 200 {
			p.Bot.Reply(e, "It's not just you! %s looks down from here.", url)
			return
		}

		p.Bot.Reply(e, "It's just you! %s looks up from here!", url)
	}()
}
