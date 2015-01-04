package linkproviders

import (
	"net/url"
	"regexp"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/plugins"
)

type RedditUser struct {
	Data struct {
		Name         string `json:"name"`
		LinkKarma    int    `json:"link_karma"`
		CommentKarma int    `json:"comment_karma"`
		IsGold       bool   `json:"is_gold"`
		IsMod        bool   `json:"is_mod"`
	} `json:"data"`
}

type RedditSub struct {
	Data struct {
		Url         string `json:"url"`
		Subscribers int    `json:"subscribers"`
		Description string `json:"public_description"`
		Actives     int    `json:"accounts_active"`
	} `json:"data"`
}

type RedditComment struct {
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

func init() {
	bot.RegisterPlugin("linkprovider:reddit", NewRedditProvider)
}

func NewRedditProvider(p *plugins.URLPlugin) error {
	p.Register("www.reddit.com", HandleReddit)
	return nil
}

func HandleReddit(c *irc.Client, e *irc.Event, u *url.URL) bool {
	if redditUserRegex.MatchString(u.Path) {
		return redditGetUser(c, e, u.Path)
	} else if redditCommentRegex.MatchString(u.Path) {
		return redditGetComment(c, e, u.Path)
	} else if redditSubRegex.MatchString(u.Path) {
		return redditGetSub(c, e, u.Path)
	}

	return false
}

func redditGetUser(c *irc.Client, e *irc.Event, url string) bool {
	matches := redditUserRegex.FindStringSubmatch(url)
	if len(matches) != 3 {
		return false
	}

	ru := &RedditUser{}
	err := JsonRequest(ru, "https://www.reddit.com/user/%s/about.json", matches[2])
	if err != nil {
		return false
	}

	// jsvana [gold] has 1 link karma and 1337 comment karma
	gold := ""
	if ru.Data.IsGold {
		gold = " [gold]"
	}

	c.Reply(e, "%s %s%s has %d link karma and %d comment karma", redditPrefix, ru.Data.Name, gold, ru.Data.LinkKarma, ru.Data.CommentKarma)

	return true
}

func redditGetComment(c *irc.Client, e *irc.Event, url string) bool {
	matches := redditCommentRegex.FindStringSubmatch(url)
	if len(matches) != 2 {
		return false
	}

	rc := []RedditComment{}
	err := JsonRequest(&rc, "https://www.reddit.com/comments/%s.json", matches[1])
	if err != nil || len(rc) < 1 {
		return false
	}

	cm := rc[0].Data.Children[0].Data

	// Title title - jsvana (/r/vim, score: 5)
	c.Reply(e, "%s %s - %s (/r/%s, score: %d)", redditPrefix, cm.Title, cm.Author, cm.Subreddit, cm.Score)

	return true
}

func redditGetSub(c *irc.Client, e *irc.Event, url string) bool {
	matches := redditSubRegex.FindStringSubmatch(url)
	if len(matches) != 2 {
		return false
	}

	rs := &RedditSub{}
	err := JsonRequest(rs, "https://www.reddit.com/r/%s/about.json", matches[1])
	if err != nil {
		return false
	}

	// /r/vim - Description description (1 subscriber, 2 actives)
	c.Reply(e, "%s %s - %s (%s, %s)", redditPrefix, rs.Data.Url, rs.Data.Description, lazyPluralize(rs.Data.Subscribers, "subscriber"), lazyPluralize(rs.Data.Actives, "active"))

	return true
}
