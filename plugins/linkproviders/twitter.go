package linkproviders

import (
	"net/url"
	"regexp"
	"strconv"

	"github.com/ChimeraCoder/anaconda"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/plugins"
)

type TwitterConfig struct {
	ConsumerKey       string
	ConsumerSecret    string
	AccessToken       string
	AccessTokenSecret string
}

type TwitterProvider struct {
	api *anaconda.TwitterApi
}

var twitterStatusRegex = regexp.MustCompile(`^/.*?/status/(.+)$`)
var twitterUserRegex = regexp.MustCompile(`^/([^/]+)$`)
var twitterPrefix = "[Twitter]"

func init() {
	bot.RegisterPlugin("linkprovider:twitter", NewTwitterProvider)
}

func NewTwitterProvider(b *bot.Bot, p *plugins.URLPlugin) error {
	t := &TwitterProvider{}

	tc := &TwitterConfig{}
	err := b.Config("twitter", tc)
	if err != nil {
		return err
	}

	anaconda.SetConsumerKey(tc.ConsumerKey)
	anaconda.SetConsumerSecret(tc.ConsumerSecret)
	t.api = anaconda.NewTwitterApi(tc.AccessToken, tc.AccessTokenSecret)

	p.Register("twitter.com", t.Handle)

	return nil
}

func (t *TwitterProvider) Handle(c *irc.Client, e *irc.Event, u *url.URL) bool {
	if twitterUserRegex.MatchString(u.Path) {
		return t.getUser(c, e, u.Path)
	} else if twitterStatusRegex.MatchString(u.Path) {
		return t.getTweet(c, e, u.Path)
	}

	return false
}

func (t *TwitterProvider) getUser(c *irc.Client, e *irc.Event, url string) bool {
	matches := twitterUserRegex.FindStringSubmatch(url)
	if len(matches) != 2 {
		return false
	}

	user, err := t.api.GetUsersShow(matches[1], nil)
	if err != nil {
		return false
	}

	// Jay Vana (@jsvana) - Description description
	c.Reply(e, "%s %s (@%s) - %s", twitterPrefix, user.Name, user.ScreenName, user.Description)

	return true
}

func (t *TwitterProvider) getTweet(c *irc.Client, e *irc.Event, url string) bool {
	matches := twitterStatusRegex.FindStringSubmatch(url)
	if len(matches) != 2 {
		return false
	}

	id, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return false
	}

	tweet, err := t.api.GetTweet(id, nil)
	if err != nil {
		return false
	}

	// Tweet text (@jsvana)
	c.Reply(e, "%s %s (@%s)", twitterPrefix, tweet.Text, tweet.User.ScreenName)

	return true
}
