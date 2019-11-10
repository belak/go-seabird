package extra

import (
	"context"
	"errors"
	"strings"
	"unicode"

	seabird "github.com/belak/go-seabird"
	"github.com/go-xorm/xorm"
)

func init() {
	seabird.RegisterPlugin("phrases", newPhrasesPlugin)
}

type phrasesPlugin struct {
	db *xorm.Engine
}

// Phrase is an xorm model for phrases
type Phrase struct {
	ID        int64
	Name      string `xorm:"index"`
	Value     string
	Submitter string
	Deleted   bool
}

func newPhrasesPlugin(b *seabird.Bot) error {
	p := &phrasesPlugin{
		db: CtxDB(b.Context()), // TODO: ensure db plugin loaded
	}

	err := p.db.Sync(Phrase{})
	if err != nil {
		return err
	}

	cm := b.CommandMux()

	cm.Event("forget", p.forgetCallback, &seabird.HelpInfo{
		Usage:       "<key>",
		Description: "Look up a phrase",
	})

	cm.Event("get", p.getCallback, &seabird.HelpInfo{
		Usage:       "<key>",
		Description: "Look up a phrase",
	})

	cm.Event("give", p.giveCallback, &seabird.HelpInfo{
		Usage:       "<user> <key>",
		Description: "Mentions a user with a given phrase",
	})

	cm.Event("history", p.historyCallback, &seabird.HelpInfo{
		Usage:       "<key>",
		Description: "Look up history for a key",
	})

	cm.Event("set", p.setCallback, &seabird.HelpInfo{
		Usage:       "<key> <phrase>",
		Description: "Remembers a phrase",
	})

	return nil
}

func (p *phrasesPlugin) cleanedName(name string) string {
	return strings.TrimFunc(strings.ToLower(name), unicode.IsSpace)
}

func (p *phrasesPlugin) getKey(key string) (*Phrase, error) {
	out := &Phrase{Name: p.cleanedName(key)}
	if len(out.Name) == 0 {
		return nil, errors.New("No key provided")
	}

	_, err := p.db.Get(out)
	if err != nil {
		return nil, err
	} else if len(out.Value) == 0 {
		return nil, errors.New("No results for given key")
	}

	return out, nil
}

func (p *phrasesPlugin) forgetCallback(ctx context.Context, r *seabird.Request) {
	entry := Phrase{
		Name:      p.cleanedName(r.Message.Trailing()),
		Submitter: r.Message.Prefix.Name,
		Deleted:   true,
	}

	if len(entry.Name) == 0 {
		r.MentionReply("No key supplied")
		return
	}

	_, err := p.db.InsertOne(entry)
	if err != nil {
		r.MentionReply("%s", err.Error())
		return
	}

	r.MentionReply("Forgot %s", entry.Name)
}

func (p *phrasesPlugin) getCallback(ctx context.Context, r *seabird.Request) {
	row, err := p.getKey(r.Message.Trailing())
	if err != nil {
		r.MentionReply("%s", err.Error())
		return
	}

	r.MentionReply("%s", row.Value)
}

func (p *phrasesPlugin) giveCallback(ctx context.Context, r *seabird.Request) {
	split := strings.SplitN(r.Message.Trailing(), " ", 2)
	if len(split) < 2 {
		r.MentionReply("Not enough args")
		return
	}

	row, err := p.getKey(split[1])
	if err != nil {
		r.MentionReply("%s", err.Error())
		return
	}

	r.Reply("%s: %s", split[0], row.Value)
}

func (p *phrasesPlugin) historyCallback(ctx context.Context, r *seabird.Request) {
	search := &Phrase{Name: p.cleanedName(r.Message.Trailing())}
	if len(search.Name) == 0 {
		r.MentionReply("No key provided")
		return
	}

	var data []Phrase

	if err := p.db.Find(&data, search); err != nil {
		r.MentionReply("%s", err.Error())
		return
	}

	for _, entry := range data {
		if entry.Deleted {
			r.MentionReply("%s deleted by %s", search.Name, entry.Submitter)
		} else {
			r.MentionReply("%s set by %s to %s", search.Name, entry.Submitter, entry.Value)
		}
	}
}

func (p *phrasesPlugin) setCallback(ctx context.Context, r *seabird.Request) {
	split := strings.SplitN(r.Message.Trailing(), " ", 2)
	if len(split) < 2 {
		r.MentionReply("Not enough args")
		return
	}

	entry := Phrase{
		Name:      p.cleanedName(split[0]),
		Submitter: r.Message.Prefix.Name,
		Value:     split[1],
	}

	if len(entry.Name) == 0 {
		r.MentionReply("No key provided")
		return
	}

	_, err := p.db.InsertOne(entry)
	if err != nil {
		r.MentionReply("%s", err.Error())
		return
	}

	r.MentionReply("%s set to %s", entry.Name, entry.Value)
}
