package plugins

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode"

	seabird ".."
	"github.com/thoj/go-ircevent"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

func init() {
	seabird.RegisterPlugin("karma", NewKarmaPlugin)
}

type Karma struct {
	Name  string `bson:"name"`
	Score int    `bson:"score"`
}

type KarmaPlugin struct {
	Bot *seabird.Bot
	C   *mgo.Collection
}

var regex = regexp.MustCompile(`((?:\w+[\+-]?)*\w)(\+\+|--)(?:\s|$)`)

func NewKarmaPlugin(b *seabird.Bot, c json.RawMessage) {
	p := &KarmaPlugin{b, b.DB.C("karma")}
	b.RegisterFunction("karma", p.Karma)
	b.RegisterCallback("PRIVMSG", p.Msg)
}

func (p *KarmaPlugin) GetKarmaFor(name string) *Karma {
	name = strings.TrimFunc(strings.ToLower(name), unicode.IsSpace)
	k := &Karma{}
	err := p.C.Find(bson.M{"name": name}).One(k)
	if err != nil {
		return &Karma{Name: name}
	}

	return k
}

func (p *KarmaPlugin) Karma(e *irc.Event) {
	p.Bot.MentionReply(e, "%s's karma is %d", e.Message(), p.GetKarmaFor(e.Message()).Score)
}

func (p *KarmaPlugin) Msg(e *irc.Event) {
	matches := regex.FindAllStringSubmatch(e.Message(), -1)
	if len(matches) > 0 && !p.Bot.Auth.UserCan(p.Bot.GetUser(e.Nick), "karma.deny") {
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
			p.C.Upsert(bson.M{"name": name}, bson.M{"$inc": bson.M{"score": diff}})
			p.Bot.Reply(e, "%s's karma is now %d", v[1], p.GetKarmaFor(name).Score)
		}
	}
}
