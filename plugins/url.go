package plugins

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
)

func init() {
	bot.RegisterPlugin("url", NewURLPlugin)
}

// NOTE: This isn't perfect in any sense of the word, but it's pretty close
// and I don't know if it's worth the time to make it better.
var urlRegex = regexp.MustCompile(`https?://[^ ]+`)
var titleRegex = regexp.MustCompile(`(?:\s*[\r\n]+\s*)+`)

// NOTE: This nasty work is done so we ignore invalid ssl certs
var client = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
	Timeout: 5 * time.Second,
}

func NewURLPlugin(bm *irc.BasicMux, cm *mux.CommandMux) error {
	bm.Event("PRIVMSG", URLTitle)

	cm.Event("down", IsItDown, &mux.HelpInfo{
		"<website>",
		"Checks if given website is down",
	})

	return nil
}

func URLTitle(c *irc.Client, e *irc.Event) {
	for _, url := range urlRegex.FindAllString(e.Trailing(), -1) {
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
				c.Reply(e, "Title: %s", str)
			}
		}(url)
	}
}

func IsItDown(c *irc.Client, e *irc.Event) {
	go func() {
		url, err := url.Parse(strings.TrimSpace(e.Trailing()))
		if err != nil {
			c.Reply(e, "URL doesn't appear to be valid")
			return
		}

		if url.Scheme == "" {
			url.Scheme = "http"
		}

		r, err := client.Head(url.String())
		if err != nil || r.StatusCode != 200 {
			c.Reply(e, "It's not just you! %s looks down from here.", url)
			return
		}

		c.Reply(e, "It's just you! %s looks up from here!", url)
	}()
}
