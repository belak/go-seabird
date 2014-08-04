package seabird

import (
	"regexp"
	"strings"
	"unicode"

	"bitbucket.org/belak/irc"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

type Karma struct {
	Name  string `bson:"name"`
	Score int    `bson:"score"`
}

type KarmaHandler struct {
	c *mgo.Collection
}

var regex = regexp.MustCompile(`((?:\w+[\+-]?)*\w)(\+\+|--)(?:\s|$)`)

func NewKarmaHandler(c *mgo.Collection) *KarmaHandler {
	return &KarmaHandler{
		c,
	}
}

func (h *KarmaHandler) GetKarmaFor(name string) *Karma {
	name = strings.TrimFunc(strings.ToLower(name), unicode.IsSpace)
	k := &Karma{}
	err := h.c.Find(bson.M{"name": name}).One(k)
	if err != nil {
		return &Karma{Name: name}
	}

	return k
}

func (h *KarmaHandler) Karma(c *irc.Client, e *irc.Event) {
	c.MentionReply(e, "%s's karma is %d", e.Trailing(), h.GetKarmaFor(e.Trailing()).Score)
}

func (h *KarmaHandler) Msg(c *irc.Client, e *irc.Event) {
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
			h.c.Upsert(bson.M{"name": name}, bson.M{"$inc": bson.M{"score": diff}})
			c.Reply(e, "%s's karma is now %d", v[1], h.GetKarmaFor(name).Score)
		}
	}
}
