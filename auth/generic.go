// +build ignore

package auth

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"strings"

	"github.com/thoj/go-ircevent"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

// TODO: Log actions
// TODO: Use trim a bit

func init() {
	seabird.RegisterAuthPlugin("generic", NewGenericAuthPlugin)
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
	p := &GenericAuthPlugin{Bot: b, C: b.DB.C("generic_auth_accounts")}
	b.RegisterFunction("login", p.Login)
	b.RegisterFunction("logout", p.Logout)
	b.RegisterFunction("register", p.Register)
	b.RegisterFunction("addperm", p.AddPerm)
	b.RegisterFunction("delperm", p.DelPerm)
	b.RegisterFunction("whois", p.Whois)
	b.RegisterCallback("JOIN", p.join)

	json.Unmarshal(c, &p.Config)

	return p
}

func (p *GenericAuthPlugin) GetHash() hash.Hash {
	h := md5.New()
	io.WriteString(h, p.Config.Salt)
	return h
}

func (p *GenericAuthPlugin) join(e *irc.Event) {
	if e.Nick == p.Bot.Conn.GetNick() {
		p.Bot.Conn.SendRawf("WHO %s", e.Message())
	}
}

func (p *GenericAuthPlugin) Whois(e *irc.Event) {
	u := p.Bot.GetUser(e.Message())
	if u.Account != "" {
		p.Bot.MentionReply(e, "%s is logged in as %s", u.CurrentNick, u.Account)
	} else {
		p.Bot.MentionReply(e, "%s is not logged in", u.CurrentNick)
	}
}

func (p *GenericAuthPlugin) Login(e *irc.Event) {
	// TODO: Don't let them login in a channel
	u := p.Bot.GetUser(e.Nick)
	if u.Account != "" {
		p.Bot.MentionReply(e, "you are already logged in")
		return
	}

	args := strings.SplitN(e.Message(), " ", 2)
	if len(args) != 2 {
		p.Bot.MentionReply(e, "usage: !login username password")
		return
	}

	h := p.GetHash()
	io.WriteString(h, args[1])

	c, err := p.C.Find(bson.M{
		"name":     args[0],
		"password": hex.EncodeToString(h.Sum(nil)),
	}).Count()

	if err != nil {
		fmt.Println(err)
		return
	}

	if c > 0 {
		u.Account = args[0]
		p.Bot.MentionReply(e, "you are now logged in as %s", args[0])
	} else {
		p.Bot.MentionReply(e, "login failed")
	}
}

func (p *GenericAuthPlugin) Logout(e *irc.Event) {
	u := p.Bot.GetUser(e.Nick)
	if u.Account == "" {
		p.Bot.MentionReply(e, "you are not logged in")
		return
	}

	u.Account = ""

	p.Bot.MentionReply(e, "you have been logged out")
}

func (p *GenericAuthPlugin) Register(e *irc.Event) {
	u := p.Bot.GetUser(e.Nick)
	if u.Account != "" {
		p.Bot.MentionReply(e, "you are already logged in")
		return
	}

	args := strings.SplitN(e.Message(), " ", 2)
	if len(args) < 2 {
		p.Bot.MentionReply(e, "usage: !register username password")
		return
	}

	c, err := p.C.Find(bson.M{
		"name": args[0],
	}).Count()

	if err != nil {
		fmt.Println(err)
		return
	}

	if c > 0 {
		p.Bot.MentionReply(e, "there is already a user with that name")
		return
	}

	h := p.GetHash()
	io.WriteString(h, args[1])

	err = p.C.Insert(bson.M{
		"name":     args[0],
		"password": hex.EncodeToString(h.Sum(nil)),
	})

	if err != nil {
		fmt.Println(err)
		return
	}

	p.Bot.MentionReply(e, "you have been registered")
}

func (p *GenericAuthPlugin) AddPerm(e *irc.Event) {
	u := p.Bot.GetUser(e.Nick)
	if !p.UserCan(u, "admin") && !p.UserCan(u, "generic_auth.addperm") {
		p.Bot.MentionReply(e, "you don't have permission to do that")
		return
	}

	args := strings.SplitN(e.Message(), " ", 2)
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

	args := strings.SplitN(e.Message(), " ", 2)
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
