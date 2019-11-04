package url

import (
	"fmt"
	"net/url"
	"regexp"

	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/plugins/utils"
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

var (
	redditPrefix = "[Reddit]"

	// /r/subreddit
	redditPrivmsgSubRegex = regexp.MustCompile(`(?:\s|^)/r/([^\s/]+)`)
	// /u/username
	redditPrivmsgUserRegex = regexp.MustCompile(`(?:\s|^)/(?:u|user)/([^\s/]+)`)

	// URL matches
	redditUserRegex    = regexp.MustCompile(`^/(?:u|user)/([^\s/]+)$`)
	redditCommentRegex = regexp.MustCompile(`^/r/[^/]+/comments/([^/]+)/.*$`)
	redditSubRegex     = regexp.MustCompile(`^/r/([^\s/]+)/?.*$`)
)

func newRedditProvider(m *seabird.BasicMux, urlPlugin *Plugin) {
	m.Event("PRIVMSG", redditPrivmsgCallback)
	urlPlugin.RegisterProvider("reddit.com", redditCallback)
}

func redditPrivmsgCallback(b *seabird.Bot, r *seabird.Request) {
	content := r.Message.Trailing()

	for _, matches := range redditPrivmsgSubRegex.FindAllStringSubmatch(content, -1) {
		redditGetSub(b, r, matches[1])
	}

	for _, matches := range redditPrivmsgUserRegex.FindAllStringSubmatch(content, -1) {
		redditGetUser(b, r, matches[1])
	}
}

func redditCallback(b *seabird.Bot, r *seabird.Request, u *url.URL) bool {
	text := u.Path

	//nolint:gocritic
	if matches := redditUserRegex.FindStringSubmatch(text); len(matches) == 2 {
		return redditGetUser(b, r, matches[1])
	} else if matches := redditCommentRegex.FindStringSubmatch(text); len(matches) == 2 {
		return redditGetComment(b, r, matches[1])
	} else if matches := redditSubRegex.FindStringSubmatch(text); len(matches) == 2 {
		return redditGetSub(b, r, matches[1])
	}

	return false
}

func redditGetUser(b *seabird.Bot, r *seabird.Request, text string) bool {
	ru := &redditUser{}
	if err := utils.GetJSON(fmt.Sprintf("https://www.reddit.com/user/%s/about.json", text), ru); err != nil {
		return false
	}

	// jsvana [gold] has 1 link karma and 1337 comment karma
	gold := ""
	if ru.Data.IsGold {
		gold = " [gold]"
	}

	b.Reply(r, "%s %s%s has %d link karma and %d comment karma", redditPrefix, ru.Data.Name, gold, ru.Data.LinkKarma, ru.Data.CommentKarma)

	return true
}

func redditGetComment(b *seabird.Bot, r *seabird.Request, text string) bool {
	rc := []redditComment{}
	if err := utils.GetJSON(fmt.Sprintf("https://www.reddit.com/comments/%s.json", text), rc); err != nil || len(rc) < 1 {
		return false
	}

	cm := rc[0].Data.Children[0].Data

	// Title title - jsvana (/r/vim, score: 5)
	b.Reply(r, "%s %s - %s (/r/%s, score: %d)", redditPrefix, cm.Title, cm.Author, cm.Subreddit, cm.Score)

	return true
}

func redditGetSub(b *seabird.Bot, r *seabird.Request, text string) bool {
	rs := &redditSub{}
	if err := utils.GetJSON(fmt.Sprintf("https://www.reddit.com/r/%s/about.json", text), rs); err != nil {
		return false
	}

	// /r/vim - Description description (1 subscriber, 2 actives)
	b.Reply(r, "%s %s - %s (%s %s, %s %s)",
		redditPrefix,
		rs.Data.URL,
		rs.Data.Description,
		utils.PrettifySuffix(rs.Data.Subscribers),
		utils.PluralizeWord(rs.Data.Subscribers, "subscriber"),
		utils.PrettifySuffix(rs.Data.Actives),
		utils.PluralizeWord(rs.Data.Actives, "active"))

	return true
}
