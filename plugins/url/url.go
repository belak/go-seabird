package url

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/lrstanley/girc"
	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	seabird "github.com/belak/go-seabird"
)

func init() {
	seabird.RegisterPlugin("url", newPlugin)
}

// NOTE: This isn't perfect in any sense of the word, but it's pretty close
// and I don't know if it's worth the time to make it better.
var (
	urlRegex     = regexp.MustCompile(`https?://[^ ]+`)
	newlineRegex = regexp.MustCompile(`\s*\n\s*`)
)

// NOTE: This nasty work is done so we ignore invalid ssl certs
var client = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
	Timeout: 5 * time.Second,
}

// LinkProvider is a callback to be registered with the Plugin. It
// takes the same parameters as a normal IRC callback in addition to a
// *url.URL representing the found url. It returns true if it was able
// to handle that url and false otherwise.
type LinkProvider func(c *girc.Client, e girc.Event, url *url.URL) bool

// Plugin stores all registeres URL LinkProviders
type Plugin struct {
	providers map[string][]LinkProvider
	logger    *logrus.Entry
}

func newPlugin(b *seabird.Bot, c *girc.Client) *Plugin {
	p := &Plugin{
		providers: make(map[string][]LinkProvider),
		logger:    b.GetLogger(),
	}

	c.Handlers.AddBg(girc.PRIVMSG, p.callback)
	c.Handlers.AddBg(seabird.PrefixCommand("down"), isItDownCallback)

	/*
		cm.Event("down", isItDownCallback, &seabird.HelpInfo{
			Usage:       "<website>",
			Description: "Checks if given website is down",
		})
	*/

	return p
}

// RegisterProvider registers a LinkProvider for a specific domain.
func (p *Plugin) RegisterProvider(domain string, f LinkProvider) error {
	p.providers[domain] = append(p.providers[domain], f)

	return nil
}

func (p *Plugin) callback(c *girc.Client, e girc.Event) {
	for _, rawurl := range urlRegex.FindAllString(e.Last(), -1) {
		go func(raw string) {
			u, err := url.ParseRequestURI(raw)
			if err != nil {
				return
			}

			// Strip the last character if it's a slash
			u.Path = strings.TrimRight(u.Path, "/")

			for _, provider := range p.providers[u.Host] {
				if provider(c, e, u) {
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
					if provider(c, e, u) {
						return
					}
				}
			}

			p.defaultLinkProvider(raw, c, e)
		}(rawurl)
	}
}

func (p *Plugin) defaultLinkProvider(url string, c *girc.Client, e girc.Event) bool {
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
		p.logger.WithError(err).Warn("Failed to grab URL")
		return false
	}

	// Scrape the tree for the first title node we find
	n, ok := scrape.Find(z, scrape.ByTag(atom.Title))

	// If we got a result, pull the text from it
	if ok {
		title := newlineRegex.ReplaceAllLiteralString(scrape.Text(n), " ")
		c.Cmd.Replyf(e, "Title: %s", title)
	}

	return ok
}

func isItDownCallback(c *girc.Client, e girc.Event) {
	url, err := url.Parse(e.Last())
	if err != nil {
		c.Cmd.Replyf(e, "URL doesn't appear to be valid")
		return
	}

	if url.Scheme == "" {
		url.Scheme = "http"
	}

	r, err := client.Head(url.String())
	if err != nil || r.StatusCode != 200 {
		c.Cmd.Replyf(e, "It's not just you! %s looks down from here.", url)
		return
	}

	c.Cmd.Replyf(e, "It's just you! %s looks up from here!", url)
}
