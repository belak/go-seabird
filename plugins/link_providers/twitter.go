package link_providers

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"

	"github.com/ChimeraCoder/anaconda"
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

var statusRegex = regexp.MustCompile(`^https://twitter.com/.*?/status/(.+)$`)
var userRegex = regexp.MustCompile(`^https://twitter.com/([^/]+)$`)

func NewTwitterProvider(b *bot.Bot) *TwitterProvider {
	t := &TwitterProvider{}

	tc := &TwitterConfig{}
	err := b.Config("twitter", tc)
	if err != nil {
		return nil
	}

	anaconda.SetConsumerKey(tc.ConsumerKey)
	anaconda.SetConsumerSecret(tc.ConsumerSecret)
	t.api = anaconda.NewTwitterApi(tc.AccessToken, tc.AccessTokenSecret)

	return t
}

func (t *TwitterProvider) Handles(url string) bool {
	return strings.HasPrefix(url, "https://twitter.com/")
}

func (t *TwitterProvider) Handle(url string, c *irc.Client, e *irc.Event) {
	if userRegex.MatchString(url) {
		t.getUser(url, c, e)
	} else if statusRegex.MatchString(url) {
		t.getTweet(url, c, e)
	}
}

func (t *TwitterProvider) getUser(url string, c *irc.Client, e *irc.Event) {
	matches := userRegex.FindStringSubmatch(url)
	if len(matches) != 2 {
		return
	}

	user, err := t.api.GetUsersShow(matches[1], nil)
	if err == nil {
		c.Reply(e, "[Twitter] %s (@%s) - %s", user.Name, user.ScreenName, user.Description)
	}
}

func (t *TwitterProvider) getTweet(url string, c *irc.Client, e *irc.Event) {
	matches := statusRegex.FindStringSubmatch(url)
	if len(matches) != 2 {
		return
	}

	id, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return
	}

	tweet, err := t.api.GetTweet(id, nil)
	if err == nil {
		c.Reply(e, "[Twitter] %s (@%s)", tweet.Text, tweet.User.ScreenName)
	}
}
