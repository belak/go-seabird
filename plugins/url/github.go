package url

import (
	"errors"
	"net/url"
	"regexp"
	"strconv"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"

	"github.com/belak/go-seabird"
	"github.com/belak/irc"
)

func init() {
	seabird.RegisterPlugin("url/github", newGithubProvider)
}

type githubConfig struct {
	Token string
}

type githubProvider struct {
	api *github.Client
}

var (
	githubUserRegex  = regexp.MustCompile(`^/([^/]+)$`)
	githubRepoRegex  = regexp.MustCompile(`^/([^/]+)/([^/]+)$`)
	githubIssueRegex = regexp.MustCompile(`^/([^/]+)/([^/]+)/issues/([^/]+)$`)
	githubPullRegex  = regexp.MustCompile(`^/([^/]+)/([^/]+)/pull/([^/]+)$`)
	githubGistRegex  = regexp.MustCompile(`^/([^/]+)/([^/]+)$`)

	githubPrefix = "[Github]"
)

func parseUserRepoNum(matches []string) (string, string, int, error) {
	if len(matches) != 4 {
		return "", "", 0, errors.New("Incorrect number of matches")
	}

	retInt, err := strconv.ParseInt(matches[3], 10, 32)
	if err != nil {
		return "", "", 0, err
	}

	return matches[1], matches[2], int(retInt), nil
}

func newGithubProvider(b *seabird.Bot, urlPlugin *URLPlugin) error {
	t := &githubProvider{}

	gc := &githubConfig{}
	err := b.Config("github", gc)
	if err != nil {
		return err
	}

	// Create an oauth2 client
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: gc.Token},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	// Create a github client from the oauth2 client
	t.api = github.NewClient(tc)

	urlPlugin.RegisterProvider("github.com", t.githubCallback)
	urlPlugin.RegisterProvider("gist.github.com", t.gistCallback)

	return nil
}

func (t *githubProvider) githubCallback(b *seabird.Bot, m *irc.Message, url *url.URL) bool {
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

func (t *githubProvider) gistCallback(b *seabird.Bot, m *irc.Message, url *url.URL) bool {
	if githubGistRegex.MatchString(url.Path) {
		return t.getGist(b, m, url.Path)
	}

	return false
}

// Jay Vana (@jsvana) at Facebook - Bio bio bio
var userTemplate = TemplateMustCompile("githubUser", `
{{- if .user.Name -}}
{{ .user.Name }}
{{- with .user.Login }}(@{{ . }}){{ end -}}
{{- else if .user.Login -}}
@{{ .user.Login }}
{{- end -}}
{{- with .user.Company }} at {{ . }}{{ end -}}
{{- with .user.Bio }} - {{ . }}{{ end -}}
`)

func (t *githubProvider) getUser(b *seabird.Bot, m *irc.Message, url string) bool {
	logger := b.GetLogger()

	matches := githubUserRegex.FindStringSubmatch(url)
	if len(matches) != 2 {
		return false
	}

	user, _, err := t.api.Users.Get(matches[1])
	if err != nil {
		logger.WithError(err).Error("Failed to get user from github")
		return false
	}

	return RenderRespond(
		b, m, logger, userTemplate, githubPrefix,
		map[string]interface{}{
			"user": user,
		},
	)
}

// jsvana/alfred [PHP] (forked from belak/alfred) Last pushed to 2 Jan 2015 - Description, 1 fork, 2 open issues, 4 stars
var repoTemplate = TemplateMustCompile("githubRepo", `
{{- .repo.FullName -}}
{{- with .repo.Language }} [{{ . }}]{{ end -}}
{{- if and .repo.Fork .repo.Parent }} (forked from {{ .repo.Parent.FullName }}){{ end }}
{{- with .repo.PushedAt }} Last pushed to {{ . | dateFormat "2 Jan 2006" }}{{ end }}
{{- with .repo.Description }} - {{ . }}{{ end }}
{{- with .repo.ForksCount }}, {{ . }} {{ "fork" | pluralize . }}{{ end }}
{{- with .repo.OpenIssuesCount }}, {{ . }} {{ "open issue" | pluralize . }}{{ end }}
{{- with .repo.StargazersCount }}, {{ . }} {{ "star" | pluralize . }}{{ end }}
`)

func (t *githubProvider) getRepo(b *seabird.Bot, m *irc.Message, url string) bool {
	logger := b.GetLogger()

	matches := githubRepoRegex.FindStringSubmatch(url)
	if len(matches) != 3 {
		return false
	}

	user := matches[1]
	repoName := matches[2]
	repo, _, err := t.api.Repositories.Get(user, repoName)

	if err != nil {
		logger.WithError(err).Error("Failed to get repo from github")
		return false
	}

	logger = logger.WithField("repo", repo)

	// If the repo doesn't have a name, we get outta there
	if repo.FullName == nil || *repo.FullName == "" {
		logger.Error("Invalid repo returned from github")
		return false
	}

	return RenderRespond(
		b, m, logger, repoTemplate, githubPrefix,
		map[string]interface{}{
			"repo": repo,
		},
	)
}

// Issue #42 on belak/go-seabird [open] (assigned to jsvana) - Issue title [created 2 Jan 2015]
var issueTemplate = TemplateMustCompile("githubIssue", `
Issue #{{ .issue.Number }} on {{ .user }}/{{ .repo }} [{{ .issue.State }}]
{{- with .issue.Assignee }} (assigned to {{ .Login }}){{ end }}
{{- with .issue.Title }} - {{ . }}{{ end }}
{{- with .issue.CreatedAt }} [created {{ . | dateFormat "2 Jan 2006" }}]{{ end }}
`)

func (t *githubProvider) getIssue(b *seabird.Bot, m *irc.Message, url string) bool {
	logger := b.GetLogger()

	matches := githubIssueRegex.FindStringSubmatch(url)
	user, repo, issueNum, err := parseUserRepoNum(matches)
	if err != nil {
		logger.WithError(err).Error("Failed to parse URL")
		return false
	}

	issue, _, err := t.api.Issues.Get(user, repo, issueNum)
	if err != nil {
		logger.WithError(err).Error("Failed to get issue from github")
		return false
	}

	return RenderRespond(
		b, m, logger, issueTemplate, githubPrefix,
		map[string]interface{}{
			"issue": issue,
			"user":  user,
			"repo":  repo,
		},
	)
}

// Pull request #59 on belak/go-seabird [open] - Title title title [created 4 Jan 2015], 1 commit, 4 comments, 2 changed files
var prTemplate = TemplateMustCompile("githubPRTemplate", `
Pull request #{{ .pull.Number }} on {{ .user }}/{{ .repo }} [{{ .pull.State }}]
{{- with .pull.User.Login }} created by {{ . }}{{ end }}
{{- with .pull.Title }} - {{ . }}{{ end }}
{{- with .pull.CreatedAt }} [created {{ . | dateFormat "2 Jan 2006" }}]{{ end }}
{{- with .pull.Commits }}, {{ . }} {{ "commit" | pluralize . }}{{ end }}
{{- with .pull.Comments }}, {{ . }} {{ "comment" | pluralize . }}{{ end }}
{{- with .pull.ChangedFiles }}, {{ . }} {{ "changed file" | pluralize . }}{{ end }}
`)

func (t *githubProvider) getPull(b *seabird.Bot, m *irc.Message, url string) bool {
	logger := b.GetLogger()

	matches := githubPullRegex.FindStringSubmatch(url)
	user, repo, pullNum, err := parseUserRepoNum(matches)
	if err != nil {
		logger.WithError(err).Error("Failed to parse URL")
		return false
	}

	pull, _, err := t.api.PullRequests.Get(user, repo, int(pullNum))
	if err != nil {
		logger.WithError(err).Error("Failed to get github pr")
		return false
	}

	return RenderRespond(
		b, m, logger, prTemplate, githubPrefix,
		map[string]interface{}{
			"user": user,
			"repo": repo,
			"pull": pull,
		},
	)
}

// Created 3 Jan 2015 by belak - Description description, 1 file, 3 comments
var gistTemplate = TemplateMustCompile("gist", `
Created {{ .gist.CreatedAt | dateFormat "2 Jan 2006" }}
{{- with .gist.Owner.Login }} by {{ . }}{{ end }}
{{- with .gist.Description }} - {{ . }}{{ end }}
{{- with .gist.Comments }}, {{ . }} {{ "comment" | pluralize .}}{{ end }}
`)

func (t *githubProvider) getGist(b *seabird.Bot, m *irc.Message, url string) bool {
	logger := b.GetLogger()

	matches := githubGistRegex.FindStringSubmatch(url)
	if len(matches) != 3 {
		return false
	}

	id := matches[2]
	gist, _, err := t.api.Gists.Get(id)
	if err != nil {
		logger.WithError(err).Error("Failed to get gist")
		return false
	}

	return RenderRespond(
		b, m, logger, gistTemplate, githubPrefix,
		map[string]interface{}{
			"gist": gist,
		},
	)
}
