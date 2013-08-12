// +build ignore

package auth

import (
	"github.com/thoj/go-ircevent"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

	"../seabird"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"hash"
	"io"
	"strings"

	"fmt"
)

// TODO: Log actions
// TODO: Use trim a bit

func init() {
	seabird.RegisterAuthPlugin("freenode", NewFreenodeAuthPlugin)
}

type GenericAccount struct {
	Id    bson.ObjectId `bson:"_id"`
	Name  string        `bson:"name"`
	Perms []string      `bson:"perms,omitempty"`
}

type GenericAuthPluginConfig struct {
	Salt string
}

type GenericAuthPlugin struct {
	Bot    *seabird.Bot
	C      *mgo.Collection
	Config GenericAuthPluginConfig
}

func NewGenericAuthPlugin(b *seabird.Bot, c json.RawMessage) seabird.AuthPlugin {
	p := &GenericAuthPlugin{Bot: b, C: b.DB.C("freenode_auth_accounts")}

	b.RegisterFunction("addperm", p.AddPerm)
	b.RegisterFunction("delperm", p.DelPerm)
	b.RegisterFunction("whois", p.Whois)

	json.Unmarshal(c, &p.Config)

	return p
}

func (p *GenericAuthPlugin) Whois(e *irc.Event) {
	u := p.Bot.GetUser(e.Message)
	if u.Account != "" {
		p.Bot.MentionReply(e, "%s is logged in as %s", u.CurrentNick, u.Account)
	} else {
		p.Bot.MentionReply(e, "%s is not logged in", u.CurrentNick)
	}
}

func (p *GenericAuthPlugin) AddPerm(e *irc.Event) {
	u := p.Bot.GetUser(e.Nick)
	if !p.UserCan(u, "admin") && !p.UserCan(u, "freenode_auth.addperm") {
		p.Bot.MentionReply(e, "you don't have permission to do that")
		return
	}

	args := strings.SplitN(e.Message, " ", 2)
	if len(args) < 2 {
		p.Bot.MentionReply(e, "usage: !addperm username perm")
		return
	}

	a := GenericAccount{}
	err := p.C.Find(bson.M{"name": args[0]}).One(&a)
	if err != nil {
		// NOTE: This may be another error, but I doubt it
		p.Bot.MentionReply(e, "account %s does not exist", args[0])
		return
	}

	for _, v := range a.Perms {
		if v == args[1] {
			p.Bot.MentionReply(e, "%s already has perm %s", args[0], args[1])
			return
		}
	}

	err = p.C.UpdateId(a.Id, bson.M{"$push": bson.M{"perms": args[1]}})
	if err != nil {
		// NOTE: This may be another error, but I doubt it
		p.Bot.MentionReply(e, "account %s does not exist", args[0])
		return
	}

	if err != nil {
		fmt.Println(err)
		return
	}

	p.Bot.MentionReply(e, "added %s perm to %s", args[1], args[0])
}

func (p *GenericAuthPlugin) DelPerm(e *irc.Event) {
	u := p.Bot.GetUser(e.Nick)
	if !p.UserCan(u, "admin") && !p.UserCan(u, "generic_auth.delperm") {
		p.Bot.MentionReply(e, "you don't have permission to do that")
		return
	}

	args := strings.SplitN(e.Message, " ", 2)
	if len(args) < 2 {
		p.Bot.MentionReply(e, "usage: !delperm username perm")
		return
	}

	err := p.C.Update(bson.M{"name": args[0]}, bson.M{"$pull": bson.M{"perms": args[1]}})
	if err != nil {
		// NOTE: This may be another error, but I doubt it
		p.Bot.MentionReply(e, "account %s does not exist", args[0])
		return
	}

	p.Bot.MentionReply(e, "removed %s perm from %s", args[1], args[0])
}

func (p *GenericAuthPlugin) UserCan(u *seabird.User, perm string) bool {
	if u.Account == "" {
		return false
	}

	c, err := p.C.Find(bson.M{
		"name":  u.Account,
		"perms": perm,
	}).Count()

	if err != nil {
		fmt.Println(err)
		return false
	}

	return c > 0
}
