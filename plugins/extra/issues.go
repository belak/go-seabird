package extra

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"

	seabird "github.com/belak/go-seabird"
	irc "gopkg.in/irc.v3"
)

func init() {
	seabird.RegisterPlugin("issues", newIssuesPlugin)
}

var issueTagRegex = regexp.MustCompile(`^(.+)(?: ([@#].+)){0,2}$`)

type issuesPlugin struct {
	Token       string
	DefaultRepo string
	RepoTags    map[string]string

	api *github.Client
}

func newIssuesPlugin(b *seabird.Bot, cm *seabird.CommandMux) error {
	p := &issuesPlugin{
		DefaultRepo: "belak/go-seabird",
		RepoTags: map[string]string{
			"irc":     "go-irc/irc",
			"seabird": "belak/go-seabird",
			"uno":     "belak/go-seabird-uno",
		},
	}
	err := b.Config("github", p)
	if err != nil {
		return err
	}

	for k, v := range p.RepoTags {
		if strings.Count(v, "/") != 1 {
			return fmt.Errorf("Invalid repo spec %q for key %q", v, k)
		}
	}

	// Create an oauth2 client
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: p.Token},
	)
	tc := oauth2.NewClient(context.TODO(), ts)

	// Create a github client from the oauth2 client
	p.api = github.NewClient(tc)

	cm.Event("issue", p.CreateIssue, &seabird.HelpInfo{
		Usage:       "<issue title> [#repo_tag] [@user]",
		Description: "Creates a new issue for seabird. Be nice. Abuse this and it will be removed.",
	})

	cm.Event("isearch", p.IssueSearch, &seabird.HelpInfo{
		Usage:       "<query string>",
		Description: "Search the seabird repo for issues.",
	})

	return nil
}

func (p *issuesPlugin) CreateIssue(b *seabird.Bot, m *irc.Message) {
	go func() {
		r := &github.IssueRequest{}

		// This will be what we eventually send to the server
		body := "Filed by " + m.Prefix.Name + " in " + m.Params[0]
		r.Body = &body

		title := strings.TrimSpace(m.Trailing())
		searchChars := "#@"
		targetRepo := p.DefaultRepo
		for idx := strings.LastIndexAny(title, searchChars); idx > -1; idx = strings.LastIndexAny(title, searchChars) {
			if strings.Contains(title[idx+1:], " ") {
				break
			}

			char := title[idx]
			data := title[idx+1:]
			title = strings.TrimSpace(title[:idx])

			switch char {
			case '#':
				if repoPath, ok := p.RepoTags[data]; ok {
					targetRepo = repoPath
				}
			case '@':
				r.Assignee = &data
			}

			searchChars = strings.Trim(searchChars, string(char))

			if len(searchChars) == 0 {
				break
			}
		}

		if title == "" {
			b.MentionReply(m, "Issue title required")
			return
		}

		r.Title = &title

		pathSegments := strings.SplitN(targetRepo, "/", 2)

		issue, _, err := p.api.Issues.Create(context.TODO(), pathSegments[0], pathSegments[1], r)
		if err != nil {
			b.MentionReply(m, "%s", err.Error())
			return
		}

		b.MentionReply(m, "Issue created. %s", *issue.HTMLURL)
	}()
}

func (p *issuesPlugin) IssueSearch(b *seabird.Bot, m *irc.Message) {
	hasState := false
	split := strings.Split(m.Trailing(), " ")
	for i := 0; i < len(split); i++ {
		if strings.HasPrefix(split[i], "repo:") {
			split = append(split[:i], split[i+1:]...)
		} else if strings.HasPrefix(split[i], "state:") {
			hasState = true
		}
	}

	split = append(split, []string{
		"repo:go-irc/irc",
		"repo:belak/go-seabird",
		"repo:belak/go-seabird-uno",
	}...)

	if !hasState {
		split = append(split, "state:open")
	}

	opt := &github.SearchOptions{}

	issues, _, err := p.api.Search.Issues(context.TODO(), strings.Join(split, " "), opt)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	total := 0
	if issues.Total != nil {
		total = *issues.Total
	}

	if total == 1 {
		b.MentionReply(m, "There was %d result.", total)
	} else {
		b.MentionReply(m, "There were %d results.", total)
	}

	if total > 3 {
		total = 3
	}

	for _, issue := range issues.Issues[:total] {
		b.MentionReply(m, "%s", encodeIssue(issue))
	}
}

func encodeIssue(issue github.Issue) string {
	// Issue #42 on belak/go-seabird [open] (assigned to jsvana) - Issue title [created 2 Jan 2015]
	urlparts := strings.Split(*issue.HTMLURL, "/")
	user := urlparts[len(urlparts)-4]
	repo := urlparts[len(urlparts)-3]

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
	if issue.HTMLURL != nil {
		out += " - " + *issue.HTMLURL
	}

	return out
}
