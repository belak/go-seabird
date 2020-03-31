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

func newtwitterProvider(b *seabird.Bot) error {
	err := b.EnsurePlugin("url")
	if err != nil {
		return err
	}

	bm := b.BasicMux()
	urlPlugin := CtxPlugin(b.Context())

	t := &twitterProvider{}

	tc := &twitterConfig{}
	if err := b.Config("twitter", tc); err != nil {
		return err
	}

	anaconda.SetConsumerKey(tc.ConsumerKey)
	anaconda.SetConsumerSecret(tc.ConsumerSecret)
	t.api = anaconda.NewTwitterApi(tc.AccessToken, tc.AccessTokenSecret)

	bm.Event("PRIVMSG", t.privmsg)
	urlPlugin.RegisterProvider("twitter.com", t.Handle)

	return nil
}

func (t *twitterProvider) privmsg(r *seabird.Request) {
	for _, matches := range twitterPrivmsgUserRegex.FindAllStringSubmatch(r.Message.Trailing(), -1) {
		t.getUser(r, matches[1])
	}
}

func (t *twitterProvider) Handle(r *seabird.Request, u *url.URL) bool {
	if matches := twitterUserRegex.FindStringSubmatch(u.Path); len(matches) == 2 {
		return t.getUser(r, matches[1])
	} else if matches := twitterStatusRegex.FindStringSubmatch(u.Path); len(matches) == 2 {
		return t.getTweet(r, matches[1])
	}

	return false
}

func (t *twitterProvider) getUser(r *seabird.Request, text string) bool {
	user, err := t.api.GetUsersShow(text, nil)
	if err != nil {
		return false
	}

	// Jay Vana (@jsvana) - Description description
	r.Replyf("%s %s (@%s) - %s", twitterPrefix, user.Name, user.ScreenName, user.Description)

	return true
}

func (t *twitterProvider) getTweet(r *seabird.Request, text string) bool {
	id, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return false
	}

	tweet, err := t.api.GetTweet(id, nil)
	if err != nil {
		return false
	}

	// Tweet text (@jsvana)
	r.Replyf("%s %s (@%s)", twitterPrefix, tweet.Text, tweet.User.ScreenName)

	return true
}
