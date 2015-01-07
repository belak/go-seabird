package plugins

import (
	"bytes"
	"html/template"
	"regexp"

	"github.com/jmoiron/sqlx"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
	"github.com/belak/seabird/plugins/infoproviders"
)

type MorningPlugin struct {
	db        *sqlx.DB
	providers map[string]*infoproviders.InfoProvider
}

func init() {
	bot.RegisterPlugin("morning", NewMorningPlugin)
}

func NewMorningPlugin(b *irc.BasicMux, m *mux.CommandMux, db *sqlx.DB) (*MorningPlugin, error) {
	p := &MorningPlugin{
		db,
		make(map[string]*infoproviders.InfoProvider),
	}

	m.Event("addmsg", p.AddMessage, &mux.HelpInfo{
		"<format string>",
		"Adds a message to your morning bot response",
	})
	m.Event("listmsg", p.ListMessages, &mux.HelpInfo{
		Description: "Lists all morning bot messages",
	})
	b.Event("PRIVMSG", p.Msg)

	return p, nil
}

func (p *MorningPlugin) Register(provider string, i *infoproviders.InfoProvider) error {
	p.providers[provider] = i

	return nil
}

func (p *MorningPlugin) Plugin(plug string) *infoproviders.InfoProvider {
	prov, ok := p.providers[plug]
	if !ok {
		return nil
	}

	return *prov.Get()
}

var morningRegex = regexp.MustCompile(`(?i)^(good\s)?morning`)

func (p *MorningPlugin) Msg(c *irc.Client, e *irc.Event) {
	if len(e.Args) < 2 || !e.FromChannel() {
		return
	}

	if morningRegex.MatchString(e.Args[1]) {
		p.sendMorning(c, e)
	}
}

func (p *MorningPlugin) AddMessage(c *irc.Client, e *irc.Event) {
	if len(e.Args) < 2 {
		c.MentionReply(e, "Format string required")
		return
	}

	message := e.Args[1]

	_, err := p.db.Exec("INSERT INTO morning_messages (nick, message) VALUES ($1, $2)", e.Identity.Nick, message)
	if err != nil {
		c.Writef("PRIVMSG %s :Error adding message (%s)", e.Identity.Nick, err)
		return
	}

	c.Writef("PRIVMSG %s :Message added successfully", e.Identity.Nick)
}

func (p *MorningPlugin) ListMessages(c *irc.Client, e *irc.Event) {
	var messages []string
	err := p.db.Select(&messages, "SELECT message FROM morning_messages WHERE nick=$1", e.Identity.Nick)
	if err != nil {
		c.Writef("PRIVMSG %s :Error fetching messages", e.Identity.Nick)
		return
	}

	if len(messages) == 0 {
		c.Writef("PRIVMSG %s :You have no messages, yet", e.Identity.Nick)
		return
	}

	c.Writef("PRIVMSG %s :Messages:", e.Identity.Nick)
	for _, msg := range messages {
		c.Writef("PRIVMSG %s :%s", e.Identity.Nick, msg)
	}
}

func (p *MorningPlugin) sendMorning(c *irc.Client, e *irc.Event) {
	var messages []string
	err := p.db.Select(&messages, "SELECT message FROM morning_messages WHERE nick=$1", e.Identity.Nick)
	if err != nil {
		return
	}

	// TODO: Figure out preferences obj and pass into Get
	for _, msg := range messages {
		t, err := template.New("msg").Parse(msg)
		if err != nil {
			continue
		}

		var doc bytes.Buffer
		t.Execute(&doc, p)
		s := doc.String()

		c.Writef("PRIVMSG %s :%s", e.Identity.Nick, s)
	}
}
