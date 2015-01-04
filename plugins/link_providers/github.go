package link_providers

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"

	"github.com/google/go-github/github"

	"code.google.com/p/goauth2/oauth"
)

type GithubConfig struct {
	Token string
}

type GithubProvider struct {
	api *github.Client
}

var githubUserRegex = regexp.MustCompile(`^https://github.com/([^/]+)$`)
var githubRepoRegex = regexp.MustCompile(`^https://github.com/([^/]+)/([^/]+)$`)
var githubIssueRegex = regexp.MustCompile(`^https://github.com/([^/]+)/([^/]+)/issues/([^/]+)$`)
var githubPrefix = "[Github]"

func NewGithubProvider(b *bot.Bot) *GithubProvider {
	t := &GithubProvider{}

	tc := &GithubConfig{}
	err := b.Config("github", tc)
	if err != nil {
		return nil
	}
	tr := &oauth.Transport{
		Token: &oauth.Token{AccessToken: tc.Token},
	}

	t.api = github.NewClient(tr.Client())

	return t
}

func (t *GithubProvider) Handles(url string) bool {
	return strings.HasPrefix(url, "https://github.com/")
}

func (t *GithubProvider) Handle(url string, c *irc.Client, e *irc.Event) {
	if githubUserRegex.MatchString(url) {
		t.getUser(url, c, e)
	} else if githubRepoRegex.MatchString(url) {
		t.getRepo(url, c, e)
	} else if githubIssueRegex.MatchString(url) {
		t.getIssue(url, c, e)
	}
}

func (t *GithubProvider) getUser(url string, c *irc.Client, e *irc.Event) {
	matches := githubUserRegex.FindStringSubmatch(url)
	if len(matches) != 2 {
		return
	}

	user, _, err := t.api.Users.Get(matches[1])
	if err != nil {
		return
	}

	out := ""
	if user.Name != nil && *user.Name != "" {
		out += *user.Name
		if user.Login != nil && *user.Login != "" {
			out += " (@" + *user.Login + ")"
		}
	} else {
		if user.Login != nil && *user.Login != "" {
			out += "@" + *user.Login
		} else {
			// If there's no name or login, fuggetaboutit
			return
		}
	}

	if user.Company != nil && *user.Company != "" {
		out += " at " + *user.Company
	}
	if user.Bio != nil && *user.Bio != "" {
		out += " - " + *user.Bio
	}

	c.Reply(e, "%s %s", githubPrefix, out)
}

func (t *GithubProvider) getRepo(url string, c *irc.Client, e *irc.Event) {
	matches := githubRepoRegex.FindStringSubmatch(url)
	if len(matches) != 3 {
		return
	}

	user := matches[1]
	repoName := matches[2]
	repo, _, err := t.api.Repositories.Get(user, repoName)
	// If the repo doesn't have a name, we get outta there
	if repo.FullName == nil || *repo.FullName == "" || err != nil {
		return
	}

	out := *repo.FullName
	if repo.Language != nil && *repo.Language != "" {
		out += " [" + *repo.Language + "]"
	}
	if repo.Fork != nil && *repo.Fork && repo.Parent != nil {
		out += " (forked from " + *repo.Parent.FullName + ")"
	}
	if repo.PushedAt != nil {
		out += " Last pushed to " + (*repo.PushedAt).Time.Format("2 Jan 2006")
	}
	if repo.Description != nil && *repo.Description != "" {
		out += " - " + *repo.Description
	}
	if repo.ForksCount != nil {
		out += fmt.Sprintf(", %s", lazyPluralize(*repo.ForksCount, "fork"))
	}
	if repo.OpenIssuesCount != nil {
		out += fmt.Sprintf(", %s", lazyPluralize(*repo.OpenIssuesCount, "open issue"))
	}
	if repo.StargazersCount != nil {
		out += fmt.Sprintf(", %s", lazyPluralize(*repo.StargazersCount, "star"))
	}

	c.Reply(e, "%s %s", githubPrefix, out)
}

func lazyPluralize(count int, word string) string {
	if count != 1 {
		return fmt.Sprintf("%d %s", count, word+"s")
	}

	return fmt.Sprintf("%d %s", count, word)
}

func (t *GithubProvider) getIssue(url string, c *irc.Client, e *irc.Event) {
	matches := githubIssueRegex.FindStringSubmatch(url)
	if len(matches) != 4 {
		return
	}

	user := matches[1]
	repo := matches[2]
	issueNum, err := strconv.ParseInt(matches[3], 10, 32)
	if err != nil {
		return
	}

	issue, _, err := t.api.Issues.Get(user, repo, int(issueNum))
	if err != nil {
		return
	}

	out := fmt.Sprintf("Issue #%d on %s/%s", *issue.Number, user, repo)
	if issue.Assignee != nil {
		out += " (assigned to " + *issue.Assignee.Login + ")"
	}
	if issue.Title != nil && *issue.Title != "" {
		out += " - " + *issue.Title
	}
	if issue.CreatedAt != nil {
		out += " [created " + (*issue.CreatedAt).Format("2 Jan 2006") + "]"
	}

	c.Reply(e, "%s %s", githubPrefix, out)
}
