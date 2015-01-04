package link_providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
)

type BitbucketProvider struct{}

type BitbucketUser struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type BitbucketRepo struct {
	Scm         string `json:"scm"`
	Description string `json:"description"`
	FullName    string `json:"full_name"`
	Language    string `json:"language"`
	UpdatedOn   string `json:"updated_on"`
}

type BitbucketIssue struct {
	Status       string        `json:"status"`
	Priority     string        `json:"priority"`
	Title        string        `json:"title"`
	ReportedBy   BitbucketUser `json:"reported_by"`
	CommentCount int           `json:"comment_count"`
	CreatedOn    string        `json:"created_on"`
	Metadata     struct {
		Kind string `json:"kind"`
	} `json:"metadata"`
}

type BitbucketPullRequest struct {
	State        string        `json:"state"`
	Title        string        `json:"title"`
	Author       BitbucketUser `json:"author"`
	CommentCount int           `json:"comment_count"`
	CreatedOn    string        `json:"created_on"`
}

var bitbucketUserRegex = regexp.MustCompile(`^https://bitbucket.org/([^/]+)$`)
var bitbucketRepoRegex = regexp.MustCompile(`^https://bitbucket.org/([^/]+)/([^/]+)$`)
var bitbucketIssueRegex = regexp.MustCompile(`^https://bitbucket.org/([^/]+)/([^/]+)/issue/([^/]+)/[^/]+$`)
var bitbucketPullRegex = regexp.MustCompile(`^https://bitbucket.org/([^/]+)/([^/]+)/pull-request/([^/]+)/.*$`)
var bitbucketPrefix = "[Bitbucket]"

func NewBitbucketProvider(_ *bot.Bot) *BitbucketProvider {
	t := &BitbucketProvider{}

	return t
}

func (t *BitbucketProvider) Handle(url string, c *irc.Client, e *irc.Event) bool {
	if bitbucketUserRegex.MatchString(url) {
		return t.getUser(url, c, e)
	} else if bitbucketRepoRegex.MatchString(url) {
		return t.getRepo(url, c, e)
	} else if bitbucketIssueRegex.MatchString(url) {
		return t.getIssue(url, c, e)
	} else if bitbucketPullRegex.MatchString(url) {
		return t.getPull(url, c, e)
	}

	return false
}

func (t *BitbucketProvider) getUser(url string, c *irc.Client, e *irc.Event) bool {
	matches := bitbucketUserRegex.FindStringSubmatch(url)
	if len(matches) != 2 {
		return false
	}

	resp, err := http.Get("https://bitbucket.org/api/2.0/users/" + matches[1])
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	bu := &BitbucketUser{}
	err = json.NewDecoder(resp.Body).Decode(bu)
	if err != nil {
		return false
	}

	c.Reply(e, "%s %s (@%s)", bitbucketPrefix, bu.DisplayName, bu.Username)

	return true
}

func (t *BitbucketProvider) getRepo(url string, c *irc.Client, e *irc.Event) bool {
	matches := bitbucketRepoRegex.FindStringSubmatch(url)
	if len(matches) != 3 {
		return false
	}

	user := matches[1]
	repo := matches[2]
	resp, err := http.Get("https://bitbucket.org/api/2.0/repositories/" + user + "/" + repo)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	br := &BitbucketRepo{}
	err = json.NewDecoder(resp.Body).Decode(br)
	if err != nil {
		return false
	}

	// chriskempson/base16-iterm2 [Shell] Last pushed to 15 Nov 2014 - Base16 for iTerm2
	out := br.FullName
	if br.Language != "" {
		out += " [" + br.Language + "]"
	}
	tm, err := time.Parse(time.RFC3339, br.UpdatedOn)
	if err != nil {
		return false
	}
	out += " Last pushed to " + tm.Format("2 Jan 2006")
	c.Reply(e, "%s %s", bitbucketPrefix, out)

	return true
}

func (t *BitbucketProvider) getIssue(url string, c *irc.Client, e *irc.Event) bool {
	matches := bitbucketIssueRegex.FindStringSubmatch(url)
	if len(matches) != 4 {
		return false
	}

	user := matches[1]
	repo := matches[2]
	issueNum := matches[3]
	uri := fmt.Sprintf("https://bitbucket.org/api/1.0/repositories/%s/%s/issues/%s", user, repo, issueNum)
	resp, err := http.Get(uri)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	bi := &BitbucketIssue{}
	err = json.NewDecoder(resp.Body).Decode(bi)
	if err != nil {
		return false
	}

	// Issue #51 on belak/seabird - Expand issues plugin with more of Bitbucket [created 3 Jan 2015]
	out := fmt.Sprintf("Issue #%s on %s/%s [%s]", issueNum, user, repo, bi.Status)
	if bi.Priority != "" && bi.Metadata.Kind != "" {
		out += " [" + bi.Priority + " - " + bi.Metadata.Kind + "]"
	}
	out += " by " + bi.ReportedBy.Username
	if bi.Title != "" {
		out += " - " + bi.Title
	}
	tm, err := time.Parse("2006-01-02T15:04:05.000", bi.CreatedOn)
	if err != nil {
		return false
	}
	out += " [created " + tm.Format("2 Jan 2006") + "]"
	c.Reply(e, "%s %s", bitbucketPrefix, out)

	return true
}

func (t *BitbucketProvider) getPull(url string, c *irc.Client, e *irc.Event) bool {
	matches := bitbucketPullRegex.FindStringSubmatch(url)
	if len(matches) != 4 {
		return false
	}

	user := matches[1]
	repo := matches[2]
	pullNum := matches[3]
	uri := fmt.Sprintf("https://bitbucket.org/api/2.0/repositories/%s/%s/pullrequests/%s", user, repo, pullNum)
	resp, err := http.Get(uri)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	bpr := &BitbucketPullRequest{}
	err = json.NewDecoder(resp.Body).Decode(bpr)
	if err != nil {
		return false
	}

	// Pull request #59 on belak/seabird created by jsvana - Add stuff to links [created 4 Jan 2015]
	out := fmt.Sprintf("Pull request #%s on %s/%s created by %s [%s]", pullNum, user, repo, bpr.Author.Username, strings.ToLower(bpr.State))
	if bpr.Title != "" {
		out += " - " + bpr.Title
	}
	tm, err := time.Parse("2006-01-02T15:04:05.000000-07:00", bpr.CreatedOn)
	if err != nil {
		return false
	}
	out += " [created " + tm.Format("2 Jan 2006") + "]"
	c.Reply(e, "%s %s", bitbucketPrefix, out)

	return true
}
