// +build ignore

package plugins

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

	seabird ".."
	irc "github.com/thoj/go-ircevent"

	"encoding/json"
	"errors"
	"log"
	"strings"
)

// TODO: Replace any special characters with nothing when inserting and querying
// TODO: Add way to remove old entries
// TODO: Add way to list old entries

// TODO: Write ForgetCallback
// TODO: Write ListCallback

func init() {
	seabird.RegisterPlugin("phrases", NewPhrasesPlugin)
}

type Phrase struct {
	Active  bool   `bson:"active"`
	Name    string `bson:"name"`
	Data    string `bson:"data"`
	Version int    `bson:"version"`
}

type PhrasesPlugin struct {
	Bot *seabird.Bot
	C   *mgo.Collection
}

func NewPhrasesPlugin(b *seabird.Bot, d json.RawMessage) {
	p := &PhrasesPlugin{b, b.DB.C("phrases")}
	b.RegisterFunction("give", p.GiveCallback)
	b.RegisterFunction("get", p.GetCallback)
	b.RegisterFunction("rem", p.RememberCallback)

	//b.RegisterFunction("forget", p.ForgetCallback)
	//b.RegisterFunction("list", p.ListCallback)
}

func (p *PhrasesPlugin) FetchNewest(name string) *Phrase {
	name = strings.ToLower(name)
	phrase := &Phrase{}
	err := p.C.Find(bson.M{"name": name, "active": true}).Sort("-version").One(phrase)
	if err != nil {
		return &Phrase{Name: name, Data: "", Version: -1}
	}
	return phrase
}

func (p *PhrasesPlugin) Update(name string, data string) error {
	// Grab the original phrase
	phrase := p.FetchNewest(name)
	if data == phrase.Data {
		return errors.New("Phrase already exists with that data")
	}

	// Update it and reinsert
	phrase.Version++
	phrase.Data = data
	phrase.Active = true

	err := p.C.Insert(phrase)
	if err != nil {
		log.Printf("phrases: insert error: %v\n", err)
		return errors.New("Error while inserting")
	}

	return nil
}

func (p *PhrasesPlugin) GiveCallback(e *irc.Event) {
	args := strings.Fields(e.Message())
	if len(args) != 2 {
		return
	}
	phrase := p.FetchNewest(args[1])
	if phrase.Version == -1 {
		p.Bot.Reply(e, "Phrase does not exits")
	} else {
		p.Bot.Reply(e, "%s: %s", args[0], phrase.Data)
	}
}

func (p *PhrasesPlugin) GetCallback(e *irc.Event) {
	args := strings.Fields(e.Message())
	if len(args) != 1 {
		return
	}

	phrase := p.FetchNewest(args[0])
	if phrase.Version == -1 {
		p.Bot.Reply(e, "Phrase does not exits")
	} else {
		p.Bot.Reply(e, "%s", phrase.Data)
	}
}

func (p *PhrasesPlugin) RememberCallback(e *irc.Event) {
	// NOTE: This uses SplitN because there is no FieldsN
	args := strings.SplitN(e.Message(), " ", 2)
	if len(args) != 2 {
		return
	}

	// Clean up the args just in case
	for k := range args {
		args[k] = strings.TrimSpace(args[k])
	}

	err := p.Update(args[0], args[1])
	if err != nil {
		p.Bot.Reply(e, "%s", err.Error())
	} else {
		p.Bot.Reply(e, "phrase updated")
	}
}
