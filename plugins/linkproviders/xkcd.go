package linkproviders

import (
	"io"
	"net/http"
	"net/url"
	"regexp"

	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/plugins"
	"github.com/belak/sorcix-irc"
)

var xkcdRegex = regexp.MustCompile(`^/([^/]+)$`)
var xkcdPrefix = "[XKCD]"

type XKCDProvider struct{}

func NewXKCDProvider(p *plugins.URLPlugin) error {
	t := &XKCDProvider{}
	p.RegisterProvider("xkcd.com", t.Handle)

	return nil
}

func (p *XKCDProvider) Handle(b *bot.Bot, m *irc.Message, url *url.URL) bool {
	if url.Path != "" && !xkcdRegex.MatchString(url.Path) {
		return false
	}

	r, err := http.Get(url.String())
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
	n, ok := scrape.Find(z, scrape.ById("comic"))
	if !ok {
		return false
	}

	n, ok = scrape.Find(n, scrape.ByTag(atom.Img))
	if !ok {
		return false
	}

	b.Reply(m, "%s %s: %s", xkcdPrefix, scrape.Attr(n, "alt"), scrape.Attr(n, "title"))
	return ok
}
