package extra

import (
	"errors"
	"strings"
	"unicode"

	"github.com/go-xorm/xorm"

	seabird "github.com/belak/go-seabird"
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

func newPhrasesPlugin(b *seabird.Bot, cm *seabird.CommandMux, db *xorm.Engine) error {
	p := &phrasesPlugin{db: db}

	err := db.Sync(Phrase{})
	if err != nil {
		return err
	}

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

func (p *phrasesPlugin) forgetCallback(b *seabird.Bot, r *seabird.Request) {
	entry := Phrase{
		Name:      p.cleanedName(r.Message.Trailing()),
		Submitter: r.Message.Prefix.Name,
		Deleted:   true,
	}

	if len(entry.Name) == 0 {
		b.MentionReply(r, "No key supplied")
	}

	_, err := p.db.InsertOne(entry)
	if err != nil {
		b.MentionReply(r, "%s", err.Error())
		return
	}

	b.MentionReply(r, "Forgot %s", entry.Name)
}

func (p *phrasesPlugin) getCallback(b *seabird.Bot, r *seabird.Request) {
	row, err := p.getKey(r.Message.Trailing())
	if err != nil {
		b.MentionReply(r, "%s", err.Error())
		return
	}

	b.MentionReply(r, "%s", row.Value)
}

func (p *phrasesPlugin) giveCallback(b *seabird.Bot, r *seabird.Request) {
	split := strings.SplitN(r.Message.Trailing(), " ", 2)
	if len(split) < 2 {
		b.MentionReply(r, "Not enough args")
		return
	}

	row, err := p.getKey(split[1])
	if err != nil {
		b.MentionReply(r, "%s", err.Error())
		return
	}

	b.Reply(r, "%s: %s", split[0], row.Value)
}

func (p *phrasesPlugin) historyCallback(b *seabird.Bot, r *seabird.Request) {
	search := &Phrase{Name: p.cleanedName(r.Message.Trailing())}
	if len(search.Name) == 0 {
		b.MentionReply(r, "No key provided")
		return
	}

	var data []Phrase

	if err := p.db.Find(&data, search); err != nil {
		b.MentionReply(r, "%s", err.Error())
		return
	}

	for _, entry := range data {
		if entry.Deleted {
			b.MentionReply(r, "%s deleted by %s", search.Name, entry.Submitter)
		} else {
			b.MentionReply(r, "%s set by %s to %s", search.Name, entry.Submitter, entry.Value)
		}
	}
}

func (p *phrasesPlugin) setCallback(b *seabird.Bot, r *seabird.Request) {
	split := strings.SplitN(r.Message.Trailing(), " ", 2)
	if len(split) < 2 {
		b.MentionReply(r, "Not enough args")
		return
	}

	entry := Phrase{
		Name:      p.cleanedName(split[0]),
		Submitter: r.Message.Prefix.Name,
		Value:     split[1],
	}

	if len(entry.Name) == 0 {
		b.MentionReply(r, "No key provided")
		return
	}

	_, err := p.db.InsertOne(entry)
	if err != nil {
		b.MentionReply(r, "%s", err.Error())
		return
	}

	b.MentionReply(r, "%s set to %s", entry.Name, entry.Value)
}
