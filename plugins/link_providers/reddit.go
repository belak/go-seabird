package link_providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
)

type RedditProvider struct{}

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

var redditUserRegex = regexp.MustCompile(`^https?://www.reddit.com/(u|user)/([^/]+)$`)
var redditCommentRegex = regexp.MustCompile(`^https?://www.reddit.com/r/[^/]+/comments/([^/]+)/.*$`)
var redditSubRegex = regexp.MustCompile(`^https?://www.reddit.com/r/([^/]+)/?.*$`)
var redditPrefix = "[Reddit]"

func NewRedditProvider(_ *bot.Bot) *RedditProvider {
	t := &RedditProvider{}

	return t
}

func (t *RedditProvider) Handle(url string, c *irc.Client, e *irc.Event) bool {
	if redditUserRegex.MatchString(url) {
		return t.getUser(url, c, e)
	} else if redditCommentRegex.MatchString(url) {
		return t.getComment(url, c, e)
	} else if redditSubRegex.MatchString(url) {
		return t.getSub(url, c, e)
	}

	return false
}

func (t *RedditProvider) getUser(url string, c *irc.Client, e *irc.Event) bool {
	matches := redditUserRegex.FindStringSubmatch(url)
	if len(matches) != 3 {
		return false
	}

	user := matches[2]
	uri := fmt.Sprintf("https://www.reddit.com/user/%s/about.json", user)
	resp, err := http.Get(uri)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	ru := &RedditUser{}
	err = json.NewDecoder(resp.Body).Decode(ru)
	if err != nil {
		return false
	}

	gold := ""
	if ru.Data.IsGold {
		gold = " [gold]"
	}

	c.Reply(e, "%s %s%s has %d link karma and %d comment karma", redditPrefix, ru.Data.Name, gold, ru.Data.LinkKarma, ru.Data.CommentKarma)

	return true
}

func (t *RedditProvider) getComment(url string, c *irc.Client, e *irc.Event) bool {
	matches := redditCommentRegex.FindStringSubmatch(url)
	if len(matches) != 2 {
		return false
	}

	id := matches[1]
	uri := fmt.Sprintf("https://www.reddit.com/comments/%s.json", id)
	resp, err := http.Get(uri)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	rc := []RedditComment{}
	err = json.NewDecoder(resp.Body).Decode(&rc)
	if err != nil || len(rc) < 1 {
		return false
	}

	cm := rc[0].Data.Children[0].Data

	c.Reply(e, "%s %s - %s (/r/%s, score: %d)", redditPrefix, cm.Title, cm.Author, cm.Subreddit, cm.Score)

	return true
}

func (t *RedditProvider) getSub(url string, c *irc.Client, e *irc.Event) bool {
	matches := redditSubRegex.FindStringSubmatch(url)
	if len(matches) != 2 {
		return false
	}

	sub := matches[1]
	uri := fmt.Sprintf("https://www.reddit.com/r/%s/about.json", sub)
	resp, err := http.Get(uri)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	rs := &RedditSub{}
	err = json.NewDecoder(resp.Body).Decode(rs)
	if err != nil {
		return false
	}

	c.Reply(e, "%s %s - %s (%s, %s)", redditPrefix, rs.Data.Url, rs.Data.Description, lazyPluralize(rs.Data.Subscribers, "subscriber"), lazyPluralize(rs.Data.Actives, "active"))

	return true
}
