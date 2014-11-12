package plugins

import (
	"regexp"
	"strings"
	"unicode"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
)

func init() {
	bot.RegisterPlugin("karma", NewKarmaPlugin)
}

type Karma struct {
	Name  string `bson:"name"`
	Score int    `bson:"score"`
}

type KarmaPlugin struct {
	c *mgo.Collection
}

var regex = regexp.MustCompile(`((?:\w+[\+-]?)*\w)(\+\+|--)(?:\s|$)`)

func NewKarmaPlugin(b *bot.Bot) (bot.Plugin, error) {
	p := &KarmaPlugin{
		b.DB.C("karma"),
	}

	b.Command("karma", "[object]", p.Karma)
	b.Event("PRIVMSG", p.Msg)

	return p, nil
}

func (p *KarmaPlugin) GetKarmaFor(name string) *Karma {
	name = strings.TrimFunc(strings.ToLower(name), unicode.IsSpace)
	k := &Karma{}
	err := p.c.Find(bson.M{"name": name}).One(k)
	if err != nil {
		return &Karma{Name: name}
	}

	return k
}

func (p *KarmaPlugin) Karma(b *bot.Bot, e *irc.Event) {
	b.MentionReply(e, "%s's karma is %d", e.Trailing(), p.GetKarmaFor(e.Trailing()).Score)
}

func (p *KarmaPlugin) Msg(b *bot.Bot, e *irc.Event) {
	if len(e.Args) < 2 || !e.FromChannel() {
		return
	}

	matches := regex.FindAllStringSubmatch(e.Trailing(), -1)
	if len(matches) > 0 {
		for _, v := range matches {
			if len(v) < 3 {
				continue
			}

			var diff int
			if v[2] == "++" {
				diff = 1
			} else {
				diff = -1
			}

			name := strings.ToLower(v[1])
			if name == e.Identity.Nick {
				// penalize self-karma
				diff = -1
			}

			p.c.Upsert(bson.M{"name": name}, bson.M{"$inc": bson.M{"score": diff}})
			b.Reply(e, "%s's karma is now %d", v[1], p.GetKarmaFor(name).Score)
		}
	}
}
