package plugins

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/belak/seabird/bot"
	"github.com/belak/sorcix-irc"
)

type IssueResult struct {
	Url string `json:"html_url"`
}

type IssuesPlugin struct {
	Token string
}

func NewIssuesPlugin() bot.Plugin {
	return &IssuesPlugin{}
}

func (p *IssuesPlugin) Register(b *bot.Bot) error {
	b.Config("github", p)

	b.CommandMux.Event("issue", p.CreateIssue, &bot.HelpInfo{
		"<issue title>",
		"Creates a new issue for seabird. Be nice. Abuse this and it will be removed.",
	})

	return nil
}

func (p *IssuesPlugin) CreateIssue(b *bot.Bot, m *irc.Message) {
	go func() {
		title := m.Trailing()
		if title == "" {
			b.MentionReply(m, "Issue title required")
			return
		}

		url := "https://api.github.com/repos/belak/seabird/issues"

		hc := &http.Client{}
		params := map[string]string{
			"title": title,
			"body":  "Filed by " + m.Prefix.Name + " in " + m.Params[0],
		}
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

		ir := &IssueResult{}
		err = json.NewDecoder(resp.Body).Decode(ir)
		if err != nil {
			b.MentionReply(m, "Error reading server response")
		}

		b.MentionReply(m, "Issue created. %s", ir.Url)
	}()
}
