package link_providers

import (
	"regexp"
	"strconv"

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

func (t *TwitterProvider) Handle(url string, c *irc.Client, e *irc.Event) bool {
	if twitterUserRegex.MatchString(url) {
		return t.getUser(url, c, e)
	} else if twitterStatusRegex.MatchString(url) {
		return t.getTweet(url, c, e)
	}

	return false
}

func (t *TwitterProvider) getUser(url string, c *irc.Client, e *irc.Event) bool {
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

func (t *TwitterProvider) getTweet(url string, c *irc.Client, e *irc.Event) bool {
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
