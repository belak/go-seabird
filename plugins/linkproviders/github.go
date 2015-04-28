package linkproviders

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"

	"code.google.com/p/goauth2/oauth"
	"github.com/google/go-github/github"

	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/plugins"
	"github.com/belak/sorcix-irc"
)

type GithubConfig struct {
	Token string
}

type GithubProvider struct {
	api *github.Client
}

var githubUserRegex = regexp.MustCompile(`^/([^/]+)$`)
var githubRepoRegex = regexp.MustCompile(`^/([^/]+)/([^/]+)$`)
var githubIssueRegex = regexp.MustCompile(`^/([^/]+)/([^/]+)/issues/([^/]+)$`)
var githubPullRegex = regexp.MustCompile(`^/([^/]+)/([^/]+)/pull/([^/]+)$`)
var githubGistRegex = regexp.MustCompile(`^/([^/]+)/([^/]+)$`)
var githubPrefix = "[Github]"

func NewGithubProvider(b *bot.Bot, p *plugins.URLPlugin) error {
	t := &GithubProvider{}

	tc := &GithubConfig{}
	err := b.Config("github", tc)
	if err != nil {
		return err
	}
	tr := &oauth.Transport{
		Token: &oauth.Token{AccessToken: tc.Token},
	}

	t.api = github.NewClient(tr.Client())

	p.RegisterProvider("github.com", t.HandleGithub)
	p.RegisterProvider("gist.github.com", t.HandleGist)

	return nil
}

func (t *GithubProvider) HandleGithub(b *bot.Bot, m *irc.Message, url *url.URL) bool {
	if githubUserRegex.MatchString(url.Path) {
		return t.getUser(b, m, url.Path)
	} else if githubRepoRegex.MatchString(url.Path) {
		return t.getRepo(b, m, url.Path)
	} else if githubIssueRegex.MatchString(url.Path) {
		return t.getIssue(b, m, url.Path)
	} else if githubPullRegex.MatchString(url.Path) {
		return t.getPull(b, m, url.Path)
	}

	return false
}

func (t *GithubProvider) HandleGist(b *bot.Bot, m *irc.Message, url *url.URL) bool {
	if githubGistRegex.MatchString(url.Path) {
		return t.getGist(b, m, url.Path)
	}

	return false
}

func (t *GithubProvider) getUser(b *bot.Bot, m *irc.Message, url string) bool {
	matches := githubUserRegex.FindStringSubmatch(url)
	if len(matches) != 2 {
		return false
	}

	user, _, err := t.api.Users.Get(matches[1])
	if err != nil {
		return false
	}

	// Jay Vana (@jsvana) at Facebook - Bio bio bio
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
			return false
		}
	}

	if user.Company != nil && *user.Company != "" {
		out += " at " + *user.Company
	}
	if user.Bio != nil && *user.Bio != "" {
		out += " - " + *user.Bio
	}

	b.Reply(m, "%s %s", githubPrefix, out)

	return true
}

func (t *GithubProvider) getRepo(b *bot.Bot, m *irc.Message, url string) bool {
	matches := githubRepoRegex.FindStringSubmatch(url)
	if len(matches) != 3 {
		return false
	}

	user := matches[1]
	repoName := matches[2]
	repo, _, err := t.api.Repositories.Get(user, repoName)
	// If the repo doesn't have a name, we get outta there
	if err != nil || repo.FullName == nil || *repo.FullName == "" {
		return false
	}

	// jsvana/alfred [PHP] (forked from belak/alfred) Last pushed to 2 Jan 2015 - Description, 1 fork, 2 open issues, 4 stars
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

	b.Reply(m, "%s %s", githubPrefix, out)

	return true
}

func (t *GithubProvider) getIssue(b *bot.Bot, m *irc.Message, url string) bool {
	matches := githubIssueRegex.FindStringSubmatch(url)
	if len(matches) != 4 {
		return false
	}

	user := matches[1]
	repo := matches[2]
	issueNum, err := strconv.ParseInt(matches[3], 10, 32)
	if err != nil {
		return false
	}

	issue, _, err := t.api.Issues.Get(user, repo, int(issueNum))
	if err != nil {
		return false
	}

	// Issue #42 on belak/seabird [open] (assigned to jsvana) - Issue title [created 2 Jan 2015]
	out := fmt.Sprintf("Issue #%d on %s/%s [%s]", *issue.Number, user, repo, *issue.State)
	if issue.Assignee != nil {
		out += " (assigned to " + *issue.Assignee.Login + ")"
	}
	if issue.Title != nil && *issue.Title != "" {
		out += " - " + *issue.Title
	}
	if issue.CreatedAt != nil {
		out += " [created " + (*issue.CreatedAt).Format("2 Jan 2006") + "]"
	}

	b.Reply(m, "%s %s", githubPrefix, out)

	return true
}

func (t *GithubProvider) getPull(b *bot.Bot, m *irc.Message, url string) bool {
	matches := githubPullRegex.FindStringSubmatch(url)
	if len(matches) != 4 {
		return false
	}

	user := matches[1]
	repo := matches[2]
	pullNum, err := strconv.ParseInt(matches[3], 10, 32)
	if err != nil {
		return false
	}

	pull, _, err := t.api.PullRequests.Get(user, repo, int(pullNum))
	if err != nil {
		return false
	}

	// Pull request #59 on belak/seabird [open] - Title title title [created 3 Jan 2015], 1 commit, 4 comments, 2 changed files
	out := fmt.Sprintf("Pull request #%d on %s/%s [%s]", *pull.Number, user, repo, *pull.State)
	if pull.User != nil {
		out += " created by " + *pull.User.Login
	}
	if pull.Title != nil && *pull.Title != "" {
		out += " - " + *pull.Title
	}
	if pull.CreatedAt != nil {
		out += " [created " + (*pull.CreatedAt).Format("2 Jan 2006") + "]"
	}
	if pull.Commits != nil {
		out += fmt.Sprintf(", %s", lazyPluralize(*pull.Commits, "commit"))
	}
	if pull.Comments != nil {
		out += fmt.Sprintf(", %s", lazyPluralize(*pull.Comments, "comment"))
	}
	if pull.ChangedFiles != nil {
		out += fmt.Sprintf(", %s", lazyPluralize(*pull.ChangedFiles, "changed file"))
	}

	b.Reply(m, "%s %s", githubPrefix, out)

	return true
}

func (t *GithubProvider) getGist(b *bot.Bot, m *irc.Message, url string) bool {
	matches := githubGistRegex.FindStringSubmatch(url)
	if len(matches) != 3 {
		return false
	}

	id := matches[2]
	gist, _, err := t.api.Gists.Get(id)
	if err != nil {
		return false
	}

	// Created 3 Jan 2015 by belak - Description description, 1 file, 3 comments
	out := "Created " + (*gist.CreatedAt).Format("2 Jan 2006")
	if gist.Owner != nil {
		out += " by " + *gist.Owner.Login
	}
	if gist.Description != nil && *gist.Description != "" {
		out += " - " + *gist.Description
	}
	out += fmt.Sprintf(", %s", lazyPluralize(len(gist.Files), "file"))
	if gist.Comments != nil {
		out += fmt.Sprintf(", %s", lazyPluralize(*gist.Comments, "comment"))
	}

	b.Reply(m, "%s %s", githubPrefix, out)

	return true
}

func lazyPluralize(count int, word string) string {
	if count != 1 {
		return fmt.Sprintf("%d %s", count, word+"s")
	}

	return fmt.Sprintf("%d %s", count, word)
}
