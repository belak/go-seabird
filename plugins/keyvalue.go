package plugins

import (
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
)

func init() {
	bot.RegisterPlugin("keyvalue", NewKeyValuePlugin)
}

type KeyValuePlugin struct {
	db *sqlx.DB
}

func NewKeyValuePlugin(c *mux.CommandMux, b *irc.BasicMux, db *sqlx.DB) error {
	p := &KeyValuePlugin{
		db,
	}

	c.Event("rem", p.Add, &mux.HelpInfo{
		"<key> <value>",
		"Remember value with the given key",
	})
	c.Event("get", p.Get, &mux.HelpInfo{
		"<key>",
		"Get value of given key",
	})
	c.Event("give", p.Give, &mux.HelpInfo{
		"<user> <key>",
		"Give value of key to nick",
	})

	return nil
}

func (p *KeyValuePlugin) Add(c *irc.Client, e *irc.Event) {
	if !e.FromChannel() {
		return
	}

	idx := strings.IndexRune(e.Trailing(), ' ')
	if idx == -1 {
		c.MentionReply(e, "Value required")
		return
	}

	key := e.Trailing()[0:idx]
	value := e.Trailing()[idx+1:]

	_, err := p.db.Exec("INSERT INTO keystore VALUES ($1, $2, $3)", e.Args[0], key, value)
	if err != nil {
		c.MentionReply(e, "Error remembering (%s)", err)
		return
	}

	c.MentionReply(e, "I'll remember that.")
}

func (p *KeyValuePlugin) Get(c *irc.Client, e *irc.Event) {
	if !e.FromChannel() {
		return
	}

	var value string
	err := p.db.Get(&value, "SELECT value FROM keystore WHERE channel=$1 AND key=$2", e.Args[0], e.Trailing())
	if err != nil {
		c.MentionReply(e, "Error fetching key '%s' (%s)", e.Trailing(), err)
		return
	}

	c.Reply(e, "%s", value)
}

func (p *KeyValuePlugin) Give(c *irc.Client, e *irc.Event) {
	if !e.FromChannel() || len(e.Args) < 2 {
		return
	}

	args := strings.Split(e.Args[1], " ")

	var value string
	err := p.db.Get(&value, "SELECT value FROM keystore WHERE channel=$1 AND key=$2", e.Args[0], args[1])
	if err != nil {
		c.MentionReply(e, "Error fetching key '%s' (%s)", e.Trailing(), err)
		return
	}

	c.Reply(e, "%s: %s", args[0], value)
}
