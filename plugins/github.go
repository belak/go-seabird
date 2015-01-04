package plugins

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
)

type IssueResult struct {
	Url string `json:"html_url"`
}

type IssuesPlugin struct {
	Token string
}

func init() {
	bot.RegisterPlugin("issues", NewIssuesPlugin)
}

func NewIssuesPlugin(b *bot.Bot, m *mux.CommandMux) error {
	p := &IssuesPlugin{}

	b.Config("github", p)

	m.Event("issue", p.CreateIssue, &mux.HelpInfo{
		"<issue title>",
		"Creates a new issue for seabird. Be nice. Abuse this and it will be removed.",
	})

	return nil
}

func (p *IssuesPlugin) CreateIssue(c *irc.Client, e *irc.Event) {
	go func() {
		title := e.Trailing()
		if title == "" {
			c.MentionReply(e, "Issue title required")
			return
		}

		url := "https://api.github.com/repos/belak/seabird/issues"

		hc := &http.Client{}
		params := map[string]string{
			"title": title,
			"body":  "Filed by " + e.Identity.Nick + " in " + e.Args[0],
		}
		body, err := json.Marshal(params)
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}

		req, err := http.NewRequest("POST", url, bytes.NewReader(body))
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}

		req.Header.Add("Authorization", "token "+p.Token)
		resp, err := hc.Do(req)
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}
		defer resp.Body.Close()

		ir := &IssueResult{}
		err = json.NewDecoder(resp.Body).Decode(ir)
		if err != nil {
			c.MentionReply(e, "Error reading server response")
		}

		c.MentionReply(e, "Issue created. %s", ir.Url)
	}()
}
