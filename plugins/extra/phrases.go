package extra

import (
	"errors"
	"strings"
	"unicode"

	"github.com/belak/go-seabird"
	"github.com/belak/nut"
	"github.com/go-irc/irc"
)

func init() {
	seabird.RegisterPlugin("phrases", newPhrasesPlugin)
}

type phrasesPlugin struct {
	db *nut.DB
}

type phraseBucket struct {
	Key     string
	Entries []phrase
}

type phrase struct {
	Value     string
	Submitter string
	Deleted   bool
}

func newPhrasesPlugin(cm *seabird.CommandMux, db *nut.DB) error {
	p := &phrasesPlugin{db: db}

	err := p.db.EnsureBucket("phrases")
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

func (p *phrasesPlugin) getKey(key string) (*phrase, error) {
	row := &phraseBucket{Key: p.cleanedName(key)}
	if len(row.Key) == 0 {
		return nil, errors.New("No key provided")
	}

	err := p.db.View(func(tx *nut.Tx) error {
		bucket := tx.Bucket("phrases")
		return bucket.Get(row.Key, row)
	})

	if err != nil {
		return nil, err
	} else if len(row.Entries) == 0 {
		return nil, errors.New("No results for given key")
	}

	entry := row.Entries[len(row.Entries)-1]
	if entry.Deleted {
		return nil, errors.New("Phrase was previously deleted")
	}

	return &entry, nil
}

func (p *phrasesPlugin) forgetCallback(b *seabird.Bot, m *irc.Message) {
	row := &phraseBucket{Key: p.cleanedName(m.Trailing())}
	if len(row.Key) == 0 {
		b.MentionReply(m, "No key supplied")
	}

	entry := phrase{
		Submitter: m.Prefix.Name,
		Deleted:   true,
	}

	err := p.db.Update(func(tx *nut.Tx) error {
		bucket := tx.Bucket("phrases")
		err := bucket.Get(row.Key, row)
		if err != nil {
			return errors.New("No results for given key")
		}

		row.Entries = append(row.Entries, entry)

		return bucket.Put(row.Key, row)
	})

	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	b.MentionReply(m, "Forgot %s", row.Key)
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
	row := &phraseBucket{Key: p.cleanedName(m.Trailing())}
	if len(row.Key) == 0 {
		b.MentionReply(m, "No key provided")
		return
	}

	err := p.db.View(func(tx *nut.Tx) error {
		bucket := tx.Bucket("phrases")
		return bucket.Get(row.Key, row)
	})
	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	for _, entry := range row.Entries {
		if entry.Deleted {
			b.MentionReply(m, "%s deleted by %s", row.Key, entry.Submitter)
		} else {
			b.MentionReply(m, "%s set by %s to %s", row.Key, entry.Submitter, entry.Value)
		}
	}
}

func (p *phrasesPlugin) setCallback(b *seabird.Bot, m *irc.Message) {
	split := strings.SplitN(m.Trailing(), " ", 2)
	if len(split) < 2 {
		b.MentionReply(m, "Not enough args")
		return
	}

	row := &phraseBucket{Key: p.cleanedName(split[0])}
	if len(row.Key) == 0 {
		b.MentionReply(m, "No key provided")
		return
	}

	entry := phrase{
		Submitter: m.Prefix.Name,
		Value:     split[1],
	}

	err := p.db.Update(func(tx *nut.Tx) error {
		bucket := tx.Bucket("phrases")
		bucket.Get(row.Key, row)

		row.Entries = append(row.Entries, entry)

		return bucket.Put(row.Key, row)
	})

	if err != nil {
		b.MentionReply(m, "%s", err.Error())
		return
	}

	b.MentionReply(m, "%s set to %s", row.Key, entry.Value)
}
