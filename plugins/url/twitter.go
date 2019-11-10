package url

import (
	"net/url"
	"regexp"
	"strconv"

	"github.com/ChimeraCoder/anaconda"

	seabird "github.com/belak/go-seabird"
)

func init() {
	seabird.RegisterPlugin("url/twitter", newtwitterProvider)
}

type twitterConfig struct {
	ConsumerKey       string
	ConsumerSecret    string
	AccessToken       string
	AccessTokenSecret string
}

type twitterProvider struct {
	api *anaconda.TwitterApi
}

var (
	twitterPrefix = "[Twitter]"

	// @username
	twitterPrivmsgUserRegex = regexp.MustCompile(`(?:\s|^)@(\w+)`)

	// URL matches
	twitterStatusRegex = regexp.MustCompile(`^/.*?/status/(.+)$`)
	twitterUserRegex   = regexp.MustCompile(`^/([^/]+)$`)
)

func newtwitterProvider(b *seabird.Bot, m *seabird.BasicMux, urlPlugin *Plugin) error {
	t := &twitterProvider{}

	tc := &twitterConfig{}
	if err := b.Config("twitter", tc); err != nil {
		return err
	}

	anaconda.SetConsumerKey(tc.ConsumerKey)
	anaconda.SetConsumerSecret(tc.ConsumerSecret)
	t.api = anaconda.NewTwitterApi(tc.AccessToken, tc.AccessTokenSecret)

	m.Event("PRIVMSG", t.privmsg)
	urlPlugin.RegisterProvider("twitter.com", t.Handle)

	return nil
}

func (t *twitterProvider) privmsg(b *seabird.Bot, r *seabird.Request) {
	for _, matches := range twitterPrivmsgUserRegex.FindAllStringSubmatch(r.Message.Trailing(), -1) {
		t.getUser(b, r, matches[1])
	}
}

func (t *twitterProvider) Handle(b *seabird.Bot, r *seabird.Request, u *url.URL) bool {
	if matches := twitterUserRegex.FindStringSubmatch(u.Path); len(matches) == 2 {
		return t.getUser(b, r, matches[1])
	} else if matches := twitterStatusRegex.FindStringSubmatch(u.Path); len(matches) == 2 {
		return t.getTweet(b, r, matches[1])
	}

	return false
}

func (t *twitterProvider) getUser(b *seabird.Bot, r *seabird.Request, text string) bool {
	user, err := t.api.GetUsersShow(text, nil)
	if err != nil {
		return false
	}

	// Jay Vana (@jsvana) - Description description
	r.Reply("%s %s (@%s) - %s", twitterPrefix, user.Name, user.ScreenName, user.Description)

	return true
}

func (t *twitterProvider) getTweet(b *seabird.Bot, r *seabird.Request, text string) bool {
	id, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return false
	}

	tweet, err := t.api.GetTweet(id, nil)
	if err != nil {
		return false
	}

	// Tweet text (@jsvana)
	r.Reply("%s %s (@%s)", twitterPrefix, tweet.Text, tweet.User.ScreenName)

	return true
}
