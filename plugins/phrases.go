package plugins

import (
	"database/sql"
	"errors"
	"strings"
	"unicode"

	"github.com/belak/irc"
	"github.com/belak/go-seabird/bot"
	"github.com/jmoiron/sqlx"
)

func init() {
	bot.RegisterPlugin("phrases", NewPhrasesPlugin)
}

type PhrasesPlugin struct {
	db *sqlx.DB
}

type phrase struct {
	ID        int
	Key       string
	Value     string
	Submitter string
	Deleted   bool
}

func NewPhrasesPlugin(b *bot.Bot) (bot.Plugin, error) {
	b.LoadPlugin("db")
	p := &PhrasesPlugin{b.Plugins["db"].(*sqlx.DB)}

	b.CommandMux.Event("forget", p.forgetCallback, &bot.HelpInfo{
		Usage:       "<key>",
		Description: "Look up a phrase",
	})

	b.CommandMux.Event("get", p.getCallback, &bot.HelpInfo{
		Usage:       "<key>",
		Description: "Look up a phrase",
	})

	b.CommandMux.Event("give", p.giveCallback, &bot.HelpInfo{
		Usage:       "<key> <user>",
		Description: "Mentions a user with a given phrase",
	})

	b.CommandMux.Event("history", p.historyCallback, &bot.HelpInfo{
		Usage:       "<key>",
		Description: "Look up history for a key",
	})

	b.CommandMux.Event("set", p.setCallback, &bot.HelpInfo{
		Usage:       "<key> <phrase>",
		Description: "Remembers a phrase",
	})

	return nil, nil
}

func (p *PhrasesPlugin) cleanedName(name string) string {
	return strings.TrimFunc(strings.ToLower(name), unicode.IsSpace)
}

func (p *PhrasesPlugin) getKey(key string) (*phrase, error) {
	row := &phrase{}
	if len(key) == 0 {
		return row, errors.New("No key provided")
	}

	err := p.db.Get(row, "SELECT * FROM phrases WHERE key=$1 ORDER BY id DESC LIMIT 1", key)
	if err == sql.ErrNoRows {
		return row, errors.New("No results for given key")
	} else if err != nil {
		return row, err
	} else if row.Deleted {
		return row, errors.New("Phrase was previously deleted")
	}

	return row, nil
}

func (p *PhrasesPlugin) forgetCallback(b *bot.Bot, m *irc.Message) {
	// Ensure there is already a key for this. Note that this
	// introduces a potential race condition, but it's not super
	// important.
	name := p.cleanedName(m.Trailing())
	_, err := p.getKey(name)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	row := phrase{
		Key:       name,
		Submitter: m.Prefix.Name,
		Deleted:   true,
	}

	if len(row.Key) == 0 {
		b.MentionReply(m, "No key supplied")
		return
	}

	_, err = p.db.Exec("INSERT INTO phrases (key, submitter, deleted) VALUES ($1, $2, $3)", row.Key, row.Submitter, row.Deleted)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	b.MentionReply(m, "Forgot %s", name)
}

func (p *PhrasesPlugin) getCallback(b *bot.Bot, m *irc.Message) {
	row, err := p.getKey(m.Trailing())
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	b.MentionReply(m, "%s", row.Value)
}

func (p *PhrasesPlugin) giveCallback(b *bot.Bot, m *irc.Message) {
	split := strings.SplitN(m.Trailing(), " ", 2)
	if len(split) < 2 {
		b.MentionReply(m, "Not enough args")
		return
	}

	row, err := p.getKey(split[1])
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	b.Reply(m, "%s: %s", split[0], row.Value)
}

func (p *PhrasesPlugin) historyCallback(b *bot.Bot, m *irc.Message) {
	rows := []phrase{}
	err := p.db.Select(&rows, "SELECT * FROM phrases WHERE key=$1 ORDER BY id DESC LIMIT 5", p.cleanedName(m.Trailing()))
	if err == sql.ErrNoRows {
		b.MentionReply(m, "No results for given key")
		return
	} else if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	for _, row := range rows {
		if row.Deleted {
			b.MentionReply(m, "%s deleted by %s", row.Key, row.Submitter)
		} else {
			b.MentionReply(m, "%s set by %s to %s", row.Key, row.Submitter, row.Value)
		}
	}
}

func (p *PhrasesPlugin) setCallback(b *bot.Bot, m *irc.Message) {
	split := strings.SplitN(m.Trailing(), " ", 2)
	if len(split) < 2 {
		b.MentionReply(m, "Not enough args")
		return
	}

	row := phrase{
		Key:       p.cleanedName(split[0]),
		Submitter: m.Prefix.Name,
		Value:     split[1],
	}

	if len(row.Key) == 0 {
		b.MentionReply(m, "No key supplied")
		return
	}

	_, err := p.db.Exec("INSERT INTO phrases (key, submitter, value) VALUES ($1, $2, $3)", row.Key, row.Submitter, row.Value)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	b.MentionReply(m, "%s set to %s", row.Key, row.Value)
}
