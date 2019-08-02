package url

import (
	"io"
	"net/http"
	"net/url"
	"regexp"

	"github.com/lrstanley/girc"
	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	seabird "github.com/belak/go-seabird"
)

func init() {
	seabird.RegisterPlugin("url/xkcd", newXKCDProvider)
}

var xkcdRegex = regexp.MustCompile(`^/([^/]+)$`)
var xkcdPrefix = "[XKCD]"

func newXKCDProvider(urlPlugin *Plugin) {
	urlPlugin.RegisterProvider("xkcd.com", handleXKCD)
}

func handleXKCD(c *girc.Client, e girc.Event, url *url.URL) bool {
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

	c.Cmd.Replyf(e, "%s %s: %s", xkcdPrefix, scrape.Attr(n, "alt"), scrape.Attr(n, "title"))
	return ok
}
