package plugins

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
)

func init() {
	bot.RegisterPlugin("issues", NewIssuesPlugin)
}

type issueResult struct {
	URL string `json:"html_url"`
}

type issuesPlugin struct {
	Token string
}

func NewIssuesPlugin(b *bot.Bot) (bot.Plugin, error) {
	p := &issuesPlugin{}
	b.Config("github", p)

	b.CommandMux.Event("issue", p.CreateIssue, &bot.HelpInfo{
		Usage:       "<issue title>",
		Description: "Creates a new issue for seabird. Be nice. Abuse this and it will be removed.",
	})

	return p, nil
}

func (p *issuesPlugin) CreateIssue(b *bot.Bot, m *irc.Message) {
	go func() {
		// This will be what we eventually send to the server
		params := map[string]string{
			"body": "Filed by " + m.Prefix.Name + " in " + m.Params[0],
		}

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

			params["assignee"] = title[1:index]
			title = title[index+1:]
		}

		if title == "" {
			b.MentionReply(m, "Issue title required")
			return
		}

		params["title"] = title

		url := "https://api.github.com/repos/belak/seabird-plugins/issues"
		hc := &http.Client{}
		body, err := json.Marshal(params)
		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}

		req, err := http.NewRequest("POST", url, bytes.NewReader(body))
		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}

		req.Header.Add("Authorization", "token "+p.Token)
		resp, err := hc.Do(req)
		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}
		defer resp.Body.Close()

		ir := &issueResult{}
		err = json.NewDecoder(resp.Body).Decode(ir)
		if err != nil {
			b.MentionReply(m, "Error reading server response")
		}

		b.MentionReply(m, "Issue created. %s", ir.URL)
	}()
}
