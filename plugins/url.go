package plugins

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/belak/seabird/bot"
	"github.com/belak/sorcix-irc"
	"golang.org/x/net/html"
)

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

type URLPlugin struct {
	providers map[string][]LinkProvider
}

type LinkProvider func(b *bot.Bot, m *irc.Message, url *url.URL) bool

func NewURLPlugin() bot.Plugin {
	return &URLPlugin{
		providers: make(map[string][]LinkProvider),
	}
}

func (p *URLPlugin) Register(b *bot.Bot) error {
	b.BasicMux.Event("PRIVMSG", p.URLTitle)

	b.CommandMux.Event("down", IsItDown, &bot.HelpInfo{
		"<website>",
		"Checks if given website is down",
	})

	return nil
}

func (p *URLPlugin) RegisterProvider(domain string, f LinkProvider) error {
	p.providers[domain] = append(p.providers[domain], f)

	return nil
}

func (p *URLPlugin) URLTitle(b *bot.Bot, m *irc.Message) {
	for _, rawurl := range urlRegex.FindAllString(m.Trailing(), -1) {
		go func(raw string) {
			u, err := url.ParseRequestURI(raw)
			if err != nil {
				return
			}

			for _, provider := range p.providers[u.Host] {
				if provider(b, m, u) {
					return
				}
			}

			// If there was a www, we fall back to no www
			// This is not perfect, but it will fix a number of issues
			// Alternatively, we could require the linkifiers to
			// register multiple times
			if strings.HasPrefix(u.Host, "www.") {
				host := u.Host[4:]
				for _, provider := range p.providers[host] {
					if provider(b, m, u) {
						return
					}
				}
			}

			defaultLinkProvider(raw, b, m)
		}(rawurl)
	}
}

func defaultLinkProvider(url string, b *bot.Bot, m *irc.Message) bool {
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
		b.Reply(m, "Title: %s", str)
		return true
	} else {
		return false
	}
}

func IsItDown(b *bot.Bot, m *irc.Message) {
	go func() {
		url, err := url.Parse(m.Trailing())
		if err != nil {
			b.Reply(m, "URL doesn't appear to be valid")
			return
		}

		if url.Scheme == "" {
			url.Scheme = "http"
		}

		r, err := client.Head(url.String())
		if err != nil || r.StatusCode != 200 {
			b.Reply(m, "It's not just you! %s looks down from here.", url)
			return
		}

		b.Reply(m, "It's just you! %s looks up from here!", url)
	}()
}
