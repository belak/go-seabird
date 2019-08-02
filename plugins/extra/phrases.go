package extra

import (
	"errors"
	"strings"
	"unicode"

	"github.com/go-xorm/xorm"
	"github.com/lrstanley/girc"

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

func newPhrasesPlugin(b *seabird.Bot, c *girc.Client, db *xorm.Engine) error {
	p := &phrasesPlugin{db: db}

	err := db.Sync(Phrase{})
	if err != nil {
		return err
	}

	c.Handlers.AddBg(seabird.PrefixCommand("forget"), p.forgetCallback)
	c.Handlers.AddBg(seabird.PrefixCommand("get"), p.getCallback)
	c.Handlers.AddBg(seabird.PrefixCommand("give"), p.giveCallback)
	c.Handlers.AddBg(seabird.PrefixCommand("history"), p.historyCallback)
	c.Handlers.AddBg(seabird.PrefixCommand("set"), p.setCallback)

	/*
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
	*/

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

func (p *phrasesPlugin) forgetCallback(c *girc.Client, e girc.Event) {
	entry := Phrase{
		Name:      p.cleanedName(e.Last()),
		Submitter: e.Source.Name,
		Deleted:   true,
	}

	if len(entry.Name) == 0 {
		c.Cmd.ReplyTof(e, "No key supplied")
	}

	_, err := p.db.InsertOne(entry)
	if err != nil {
		c.Cmd.ReplyTof(e, "%s", err.Error())
		return
	}

	c.Cmd.ReplyTof(e, "Forgot %s", entry.Name)
}

func (p *phrasesPlugin) getCallback(c *girc.Client, e girc.Event) {
	row, err := p.getKey(e.Last())
	if err != nil {
		c.Cmd.ReplyTof(e, "%s", err.Error())
		return
	}

	c.Cmd.ReplyTof(e, "%s", row.Value)
}

func (p *phrasesPlugin) giveCallback(c *girc.Client, e girc.Event) {
	split := strings.SplitN(e.Last(), " ", 2)
	if len(split) < 2 {
		c.Cmd.ReplyTof(e, "Not enough args")
		return
	}

	row, err := p.getKey(split[1])
	if err != nil {
		c.Cmd.ReplyTof(e, "%s", err.Error())
		return
	}

	c.Cmd.Replyf(e, "%s: %s", split[0], row.Value)
}

func (p *phrasesPlugin) historyCallback(c *girc.Client, e girc.Event) {
	search := &Phrase{Name: p.cleanedName(e.Last())}
	if len(search.Name) == 0 {
		c.Cmd.ReplyTof(e, "No key provided")
		return
	}

	var data []Phrase
	err := p.db.Find(&data, search)
	if err != nil {
		c.Cmd.ReplyTof(e, "%s", err.Error())
		return
	}

	for _, entry := range data {
		if entry.Deleted {
			c.Cmd.ReplyTof(e, "%s deleted by %s", search.Name, entry.Submitter)
		} else {
			c.Cmd.ReplyTof(e, "%s set by %s to %s", search.Name, entry.Submitter, entry.Value)
		}
	}
}

func (p *phrasesPlugin) setCallback(c *girc.Client, e girc.Event) {
	split := strings.SplitN(e.Last(), " ", 2)
	if len(split) < 2 {
		c.Cmd.ReplyTof(e, "Not enough args")
		return
	}

	entry := Phrase{
		Name:      p.cleanedName(split[0]),
		Submitter: e.Source.Name,
		Value:     split[1],
	}

	if len(entry.Name) == 0 {
		c.Cmd.ReplyTof(e, "No key provided")
		return
	}

	_, err := p.db.InsertOne(entry)
	if err != nil {
		c.Cmd.ReplyTof(e, "%s", err.Error())
		return
	}

	c.Cmd.ReplyTof(e, "%s set to %s", entry.Name, entry.Value)
}
