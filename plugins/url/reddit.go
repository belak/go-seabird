package url

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/Unknwon/com"

	"github.com/belak/go-seabird"
	"github.com/belak/irc"
)

func init() {
	seabird.RegisterPlugin("url/reddit", newRedditProvider)
}

type redditUser struct {
	Data struct {
		Name         string `json:"name"`
		LinkKarma    int    `json:"link_karma"`
		CommentKarma int    `json:"comment_karma"`
		IsGold       bool   `json:"is_gold"`
		IsMod        bool   `json:"is_mod"`
	} `json:"data"`
}

type redditSub struct {
	Data struct {
		URL         string `json:"url"`
		Subscribers int    `json:"subscribers"`
		Description string `json:"public_description"`
		Actives     int    `json:"accounts_active"`
	} `json:"data"`
}

type redditComment struct {
	Data struct {
		Children []struct {
			Data struct {
				Title     string `json:"title"`
				Author    string `json:"author"`
				Score     int    `json:"score"`
				Subreddit string `json:"subreddit"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

var redditUserRegex = regexp.MustCompile(`^/(u|user)/([^/]+)$`)
var redditCommentRegex = regexp.MustCompile(`^/r/[^/]+/comments/([^/]+)/.*$`)
var redditSubRegex = regexp.MustCompile(`^/r/([^/]+)/?.*$`)
var redditPrefix = "[Reddit]"

func newRedditProvider(urlPlugin *Plugin) {
	urlPlugin.RegisterProvider("reddit.com", redditCallback)
}

func redditCallback(b *seabird.Bot, m *irc.Message, u *url.URL) bool {
	text := u.Path
	if redditUserRegex.MatchString(text) {
		return redditGetUser(b, m, text)
	} else if redditCommentRegex.MatchString(text) {
		return redditGetComment(b, m, text)
	} else if redditSubRegex.MatchString(text) {
		return redditGetSub(b, m, text)
	}

	return false
}

func redditGetUser(b *seabird.Bot, m *irc.Message, url string) bool {
	matches := redditUserRegex.FindStringSubmatch(url)
	if len(matches) != 3 {
		return false
	}

	ru := &redditUser{}
	err := com.HttpGetJSON(&http.Client{}, fmt.Sprintf("https://www.reddit.com/user/%s/about.json", matches[2]), ru)
	if err != nil {
		return false
	}

	// jsvana [gold] has 1 link karma and 1337 comment karma
	gold := ""
	if ru.Data.IsGold {
		gold = " [gold]"
	}

	b.Reply(m, "%s %s%s has %d link karma and %d comment karma", redditPrefix, ru.Data.Name, gold, ru.Data.LinkKarma, ru.Data.CommentKarma)

	return true
}

func redditGetComment(b *seabird.Bot, m *irc.Message, url string) bool {
	matches := redditCommentRegex.FindStringSubmatch(url)
	if len(matches) != 2 {
		return false
	}

	rc := []redditComment{}
	err := com.HttpGetJSON(&http.Client{}, fmt.Sprintf("https://www.reddit.com/comments/%s.json", matches[1]), rc)
	if err != nil || len(rc) < 1 {
		return false
	}

	cm := rc[0].Data.Children[0].Data

	// Title title - jsvana (/r/vim, score: 5)
	b.Reply(m, "%s %s - %s (/r/%s, score: %d)", redditPrefix, cm.Title, cm.Author, cm.Subreddit, cm.Score)

	return true
}

func redditGetSub(b *seabird.Bot, m *irc.Message, url string) bool {
	matches := redditSubRegex.FindStringSubmatch(url)
	if len(matches) != 2 {
		return false
	}

	rs := &redditSub{}
	err := com.HttpGetJSON(&http.Client{}, fmt.Sprintf("https://www.reddit.com/r/%s/about.json", matches[1]), rs)
	if err != nil {
		return false
	}

	// /r/vim - Description description (1 subscriber, 2 actives)
	b.Reply(m, "%s %s - %s (%s, %s)", redditPrefix, rs.Data.URL, rs.Data.Description, lazyPluralize(rs.Data.Subscribers, "subscriber"), lazyPluralize(rs.Data.Actives, "active"))

	return true
}
