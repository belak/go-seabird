package linkproviders

import (
	"net/url"
	"regexp"
	"strconv"

	"github.com/ChimeraCoder/anaconda"

	"github.com/belak/seabird/bot"
	"github.com/belak/sorcix-irc"
	"github.com/seabird-irc/plugins"
)

func init() {
	bot.RegisterPlugin("url/twitter", NewTwitterProvider)
}

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

func NewTwitterProvider(b *bot.Bot) (bot.Plugin, error) {
	// Ensure that the url plugin is loaded
	b.LoadPlugin("url")
	p := b.Plugins["url"].(*plugins.URLPlugin)

	t := &TwitterProvider{}

	tc := &TwitterConfig{}
	err := b.Config("twitter", tc)
	if err != nil {
		return nil, err
	}

	anaconda.SetConsumerKey(tc.ConsumerKey)
	anaconda.SetConsumerSecret(tc.ConsumerSecret)
	t.api = anaconda.NewTwitterApi(tc.AccessToken, tc.AccessTokenSecret)

	p.RegisterProvider("twitter.com", t.Handle)

	return nil, nil
}

func (t *TwitterProvider) Handle(b *bot.Bot, m *irc.Message, u *url.URL) bool {
	if twitterUserRegex.MatchString(u.Path) {
		return t.getUser(b, m, u.Path)
	} else if twitterStatusRegex.MatchString(u.Path) {
		return t.getTweet(b, m, u.Path)
	}

	return false
}

func (t *TwitterProvider) getUser(b *bot.Bot, m *irc.Message, url string) bool {
	matches := twitterUserRegex.FindStringSubmatch(url)
	if len(matches) != 2 {
		return false
	}

	user, err := t.api.GetUsersShow(matches[1], nil)
	if err != nil {
		return false
	}

	// Jay Vana (@jsvana) - Description description
	b.Reply(m, "%s %s (@%s) - %s", twitterPrefix, user.Name, user.ScreenName, user.Description)

	return true
}

func (t *TwitterProvider) getTweet(b *bot.Bot, m *irc.Message, url string) bool {
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
	b.Reply(m, "%s %s (@%s)", twitterPrefix, tweet.Text, tweet.User.ScreenName)

	return true
}
