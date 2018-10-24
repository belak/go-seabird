package extra

import (
	"errors"
	"strings"
	"unicode"

	"github.com/belak/go-seabird"
	"github.com/belak/nut"
	irc "github.com/go-irc/irc/v2"
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

func newPhrasesPlugin(b *seabird.Bot, cm *seabird.CommandMux, ndb *nut.DB, db *xorm.Engine) error {
	l := b.GetLogger()
	p := &phrasesPlugin{db: db}

	err := db.Sync(Phrase{})
	if err != nil {
		return err
	}

	rowCount, err := db.Count(Phrase{})
	if err != nil {
		return err
	}

	if ndb != nil && rowCount != 0 {
		l.Info("Skipping phrases migration because target table is non-empty")
	} else if ndb != nil {
		l.Info("Migrating phrases from nut to xorm")

		// This is a bit gross, but it's the simplest way to get a transaction for both nut and xorm.
		err = ndb.View(func(tx *nut.Tx) error {
			_, innerErr := p.db.Transaction(func(s *xorm.Session) (interface{}, error) {
				bucket := tx.Bucket("phrases")
				if bucket == nil {
					l.Info("Skipping phrases migration because of missing bucket")
					return nil, nil
				}

				data := &phraseBucket{}
				c := bucket.Cursor()
				for k, e := c.First(&data); e == nil; k, e = c.Next(&data) {
					l.Infof("Migrating phrase entry for %s", data.Key)

					if data.Key != k {
						l.Warnf("Phrase name (%s) does not match key (%s)", data.Key, k)
					}

					for _, entry := range data.Entries {
						phrase := Phrase{
							Name:      data.Key,
							Value:     entry.Value,
							Submitter: entry.Submitter,
							Deleted:   entry.Deleted,
						}

						_, err = s.InsertOne(phrase)
						if err != nil {
							return nil, err
						}
					}
				}

				return nil, err
			})

			return innerErr
		})
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
