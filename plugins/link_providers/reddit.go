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

func (t *RedditProvider) Handles(url string) bool {
	redditMatchRegex := regexp.MustCompile(`^https?://www.reddit.com/`)
	return redditMatchRegex.MatchString(url)
}

func (t *RedditProvider) Handle(url string, c *irc.Client, e *irc.Event) {
	if redditUserRegex.MatchString(url) {
		t.getUser(url, c, e)
	} else if redditCommentRegex.MatchString(url) {
		t.getComment(url, c, e)
	} else if redditSubRegex.MatchString(url) {
		t.getSub(url, c, e)
	}
}

func (t *RedditProvider) getUser(url string, c *irc.Client, e *irc.Event) {
	matches := redditUserRegex.FindStringSubmatch(url)
	if len(matches) != 3 {
		return
	}

	user := matches[2]
	uri := fmt.Sprintf("https://www.reddit.com/user/%s/about.json", user)
	resp, err := http.Get(uri)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	ru := &RedditUser{}
	err = json.NewDecoder(resp.Body).Decode(ru)
	if err != nil {
		return
	}

	gold := ""
	if ru.Data.IsGold {
		gold = " [gold]"
	}

	c.Reply(e, "%s %s%s has %d link karma and %d comment karma", redditPrefix, ru.Data.Name, gold, ru.Data.LinkKarma, ru.Data.CommentKarma)
}

func (t *RedditProvider) getComment(url string, c *irc.Client, e *irc.Event) {
	matches := redditCommentRegex.FindStringSubmatch(url)
	if len(matches) != 2 {
		return
	}

	id := matches[1]
	uri := fmt.Sprintf("https://www.reddit.com/comments/%s.json", id)
	resp, err := http.Get(uri)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	rc := []RedditComment{}
	err = json.NewDecoder(resp.Body).Decode(&rc)
	if err != nil || len(rc) < 1 {
		return
	}

	cm := rc[0].Data.Children[0].Data

	c.Reply(e, "%s %s - %s (/r/%s, score: %d)", redditPrefix, cm.Title, cm.Author, cm.Subreddit, cm.Score)
}

func (t *RedditProvider) getSub(url string, c *irc.Client, e *irc.Event) {
	matches := redditSubRegex.FindStringSubmatch(url)
	if len(matches) != 2 {
		return
	}

	sub := matches[1]
	uri := fmt.Sprintf("https://www.reddit.com/r/%s/about.json", sub)
	resp, err := http.Get(uri)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	rs := &RedditSub{}
	err = json.NewDecoder(resp.Body).Decode(rs)
	if err != nil {
		return
	}

	c.Reply(e, "%s %s - %s (%s, %s)", redditPrefix, rs.Data.Url, rs.Data.Description, lazyPluralize(rs.Data.Subscribers, "subscriber"), lazyPluralize(rs.Data.Actives, "active"))
}
