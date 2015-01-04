package plugins

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
	links "github.com/belak/seabird/plugins/link_providers"
)

func init() {
	bot.RegisterPlugin("url", NewURLPlugin)
}

type URLPlugin struct {
	Providers []links.LinkProvider
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

func NewURLPlugin(b *bot.Bot, bm *irc.BasicMux, cm *mux.CommandMux) error {
	p := &URLPlugin{}
	p.Providers = []links.LinkProvider{
		links.NewBitbucketProvider(b),
		links.NewGithubProvider(b),
		links.NewRedditProvider(b),
		links.NewTwitterProvider(b),

		// Must be last. DefaultProvider.Handles always returns true.
		links.NewDefaultProvider(b),
	}

	bm.Event("PRIVMSG", p.URLTitle)

	cm.Event("down", IsItDown, &mux.HelpInfo{
		"<website>",
		"Checks if given website is down",
	})

	return nil
}

func (p *URLPlugin) URLTitle(c *irc.Client, e *irc.Event) {
	for _, url := range urlRegex.FindAllString(e.Trailing(), -1) {
		go func(url string) {
			for _, provider := range p.Providers {
				if provider.Handle(url, c, e) {
					return
				}
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
