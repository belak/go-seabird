package plugins

import (
	"fmt"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
)

func init() {
	bot.RegisterPlugin("issues", NewIssuesPlugin)
}

type issuesPlugin struct {
	Token string

	api *github.Client
}

func NewIssuesPlugin(b *bot.Bot) (bot.Plugin, error) {
	p := &issuesPlugin{}
	b.Config("github", p)

	// Create an oauth2 client
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: p.Token},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	// Create a github client from the oauth2 client
	p.api = github.NewClient(tc)

	b.CommandMux.Event("issue", p.CreateIssue, &bot.HelpInfo{
		Usage:       "<issue title>",
		Description: "Creates a new issue for seabird. Be nice. Abuse this and it will be removed.",
	})
	b.CommandMux.Event("isearch", p.IssueSearch, &bot.HelpInfo{
		Usage:       "<query string>",
		Description: "Search the seabird repo for issues.",
	})

	return p, nil
}

func (p *issuesPlugin) CreateIssue(b *bot.Bot, m *irc.Message) {
	go func() {
		r := &github.IssueRequest{}

		// This will be what we eventually send to the server
		body := "Filed by " + m.Prefix.Name + " in " + m.Params[0]
		r.Body = &body

		// If the first character is an @, we assume it's a
		// user so we grab it and update what we're setting
		// the title to.
		title := m.Trailing()
		if strings.HasPrefix(title, "@") {
			index := strings.Index(title, " ")
			if index == -1 {
				b.MentionReply(m, "Issue title required")
				return
			}

			asignee := title[1:index]
			r.Assignee = &asignee

			title = title[index+1:]
		}

		if title == "" {
			b.MentionReply(m, "Issue title required")
			return
		}

		r.Title = &title

		issue, _, err := p.api.Issues.Create("belak", "seabird", r)
		if err != nil {
			b.MentionReply(m, "%s", err.Error())
			return
		}

		b.MentionReply(m, "Issue created. %s", *issue.HTMLURL)
	}()
}

func (p *issuesPlugin) IssueSearch(b *bot.Bot, m *irc.Message) {
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
		"repo:belak/irc",
		"repo:belak/seabird",
	}...)

	if !hasState {
		split = append(split, "state:open")
	}

	opt := &github.SearchOptions{}

	issues, _, err := p.api.Search.Issues(strings.Join(split, " "), opt)
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

	for i := 0; i < len(issues.Issues) && i < 3; i++ {
		issue := issues.Issues[i]
		urlparts := strings.Split(*issue.HTMLURL, "/")
		user := urlparts[len(urlparts)-4]
		repo := urlparts[len(urlparts)-3]

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
		if issue.HTMLURL != nil {
			out += " - " + *issue.HTMLURL
		}

		b.MentionReply(m, "%s", out)
	}

}
