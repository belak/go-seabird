package extra

import (
	"errors"
	"strings"
	"unicode"

	seabird "github.com/belak/go-seabird"
	"github.com/go-xorm/xorm"
	irc "gopkg.in/irc.v3"
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

// phraseBucket is the old nut.DB phrase store, along with phrase
type phraseBucket struct {
	Key     string
	Entries []phrase
}

type phrase struct {
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

func (p *phrasesPlugin) forgetCallback(b *seabird.Bot, m *irc.Message) {
	entry := Phrase{
		Name:      p.cleanedName(m.Trailing()),
		Submitter: m.Prefix.Name,
		Deleted:   true,
	}

	if len(entry.Name) == 0 {
		b.MentionReply(m, "No key supplied")
	}

	_, err := p.db.InsertOne(entry)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	b.MentionReply(m, "Forgot %s", entry.Name)
}

func (p *phrasesPlugin) getCallback(b *seabird.Bot, m *irc.Message) {
	row, err := p.getKey(m.Trailing())
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	b.MentionReply(m, "%s", row.Value)
}

func (p *phrasesPlugin) giveCallback(b *seabird.Bot, m *irc.Message) {
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

func (p *phrasesPlugin) historyCallback(b *seabird.Bot, m *irc.Message) {
	search := &Phrase{Name: p.cleanedName(m.Trailing())}
	if len(search.Name) == 0 {
		b.MentionReply(m, "No key provided")
		return
	}

	var data []Phrase
	err := p.db.Find(&data, search)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	for _, entry := range data {
		if entry.Deleted {
			b.MentionReply(m, "%s deleted by %s", search.Name, entry.Submitter)
		} else {
			b.MentionReply(m, "%s set by %s to %s", search.Name, entry.Submitter, entry.Value)
		}
	}
}

func (p *phrasesPlugin) setCallback(b *seabird.Bot, m *irc.Message) {
	split := strings.SplitN(m.Trailing(), " ", 2)
	if len(split) < 2 {
		b.MentionReply(m, "Not enough args")
		return
	}

	entry := Phrase{
		Name:      p.cleanedName(split[0]),
		Submitter: m.Prefix.Name,
		Value:     split[1],
	}

	if len(entry.Name) == 0 {
		b.MentionReply(m, "No key provided")
		return
	}

	_, err := p.db.InsertOne(entry)
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	b.MentionReply(m, "%s set to %s", entry.Name, entry.Value)
}
