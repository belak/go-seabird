package url

import (
	"net/url"
	"regexp"
	"strconv"

	"github.com/ChimeraCoder/anaconda"

	"github.com/belak/go-seabird"
	irc "github.com/go-irc/irc"
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

var twitterStatusRegex = regexp.MustCompile(`^/.*?/status/(.+)$`)
var twitterUserRegex = regexp.MustCompile(`^/([^/]+)$`)
var twitterPrefix = "[Twitter]"

func newtwitterProvider(b *seabird.Bot, urlPlugin *Plugin) error {
	t := &twitterProvider{}

	tc := &twitterConfig{}
	err := b.Config("twitter", tc)
	if err != nil {
		return err
	}

	anaconda.SetConsumerKey(tc.ConsumerKey)
	anaconda.SetConsumerSecret(tc.ConsumerSecret)
	t.api = anaconda.NewTwitterApi(tc.AccessToken, tc.AccessTokenSecret)

	urlPlugin.RegisterProvider("twitter.com", t.Handle)

	return nil
}

func (t *twitterProvider) Handle(b *seabird.Bot, m *irc.Message, u *url.URL) bool {
	if twitterUserRegex.MatchString(u.Path) {
		return t.getUser(b, m, u.Path)
	} else if twitterStatusRegex.MatchString(u.Path) {
		return t.getTweet(b, m, u.Path)
	}

	return false
}

func (t *twitterProvider) getUser(b *seabird.Bot, m *irc.Message, url string) bool {
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

func (t *twitterProvider) getTweet(b *seabird.Bot, m *irc.Message, url string) bool {
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
