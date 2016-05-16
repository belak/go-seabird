package plugins

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/belak/irc"
	"github.com/belak/go-seabird/bot"
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

// LinkProvider is a callback to be registered with the URLPlugin. It
// takes the same parameters as a normal IRC callback in addition to a
// *url.URL representing the found url. It returns true if it was able
// to handle that url and false otherwise.
type LinkProvider func(b *bot.Bot, m *irc.Message, url *url.URL) bool

// URLPlugin stores all registeres URL LinkProviders
type URLPlugin struct {
	providers map[string][]LinkProvider
}

func NewURLPlugin(b *bot.Bot) (bot.Plugin, error) {
	p := &URLPlugin{
		providers: make(map[string][]LinkProvider),
	}

	b.BasicMux.Event("PRIVMSG", p.callback)

	b.CommandMux.Event("down", isItDownCallback, &bot.HelpInfo{
		Usage:       "<website>",
		Description: "Checks if given website is down",
	})

	return p, nil
}

// RegisterProvider registers a LinkProvider for a specific domain.
func (p *URLPlugin) RegisterProvider(domain string, f LinkProvider) error {
	p.providers[domain] = append(p.providers[domain], f)

	return nil
}

func (p *URLPlugin) callback(b *bot.Bot, m *irc.Message) {
	for _, rawurl := range urlRegex.FindAllString(m.Trailing(), -1) {
		go func(raw string) {
			u, err := url.ParseRequestURI(raw)
			if err != nil {
				return
			}

			// Strip the last character if it's a slash
			u.Path = strings.TrimRight(u.Path, "/")

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

	// Scrape the tree for the first title node we find
	n, ok := scrape.Find(z, scrape.ByTag(atom.Title))

	// If we got a result, pull the text from it
	if ok {
		b.Reply(m, "Title: %s", scrape.Text(n))
	}

	return ok
}

func isItDownCallback(b *bot.Bot, m *irc.Message) {
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
