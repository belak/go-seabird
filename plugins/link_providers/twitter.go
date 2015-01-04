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

var twitterStatusRegex = regexp.MustCompile(`^https://twitter.com/.*?/status/(.+)$`)
var twitterUserRegex = regexp.MustCompile(`^https://twitter.com/([^/]+)$`)
var twitterPrefix = "[Twitter]"

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
	if twitterUserRegex.MatchString(url) {
		t.getUser(url, c, e)
	} else if twitterStatusRegex.MatchString(url) {
		t.getTweet(url, c, e)
	}
}

func (t *TwitterProvider) getUser(url string, c *irc.Client, e *irc.Event) {
	matches := twitterUserRegex.FindStringSubmatch(url)
	if len(matches) != 2 {
		return
	}

	user, err := t.api.GetUsersShow(matches[1], nil)
	if err == nil {
		c.Reply(e, "%s %s (@%s) - %s", twitterPrefix, user.Name, user.ScreenName, user.Description)
	}
}

func (t *TwitterProvider) getTweet(url string, c *irc.Client, e *irc.Event) {
	matches := twitterStatusRegex.FindStringSubmatch(url)
	if len(matches) != 2 {
		return
	}

	id, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return
	}

	tweet, err := t.api.GetTweet(id, nil)
	if err == nil {
		c.Reply(e, "%s %s (@%s)", twitterPrefix, tweet.Text, tweet.User.ScreenName)
	}
}
