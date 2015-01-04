package link_providers

import (
	"crypto/tls"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
)

type DefaultProvider struct{}

func NewDefaultProvider(_ *bot.Bot) *DefaultProvider {
	t := &DefaultProvider{}

	return t
}

func (t *DefaultProvider) Handle(url string, c *irc.Client, e *irc.Event) bool {
	var client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 5 * time.Second,
	}

	r, err := client.Get(url)
	if err != nil {
		return false
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		return false
	}

	// We search the first 1K and if a title isn't in there, we deal with it
	z, err := html.Parse(io.LimitReader(r.Body, 1024*1024))
	if err != nil {
		return false
	}

	var titleRegex = regexp.MustCompile(`(?:\s*[\r\n]+\s*)+`)

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
		// Title: title title
		c.Reply(e, "Title: %s", str)
		return true
	} else {
		return false
	}
}
