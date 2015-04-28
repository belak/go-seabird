// +build ignore

package auth

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"strings"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

	"github.com/belak/seabird/bot"
	"github.com/belak/sorcix-irc"
)

func init() {
	bot.RegisterAuthPlugin("generic", NewGenericAuthPlugin)
}

type genericAccount struct {
	Id    bson.ObjectId `bson:"_id"`
	Name  string        `bson:"name"`
	Perms []string      `bson:"perms,omitempty"`
}

type user struct {
	CurrentNick string
	Account     string
	Channels    []string
}

type GenericAuth struct {
	C     *mgo.Collection
	Salt  string
	users map[string]*user
}

func (au *GenericAuth) userCan(u *user, p string) bool {
	if u.Account == "" {
		return false
	}

	c, err := au.C.Find(bson.M{
		"name":  u.Account,
		"perms": p,
	}).Count()

	if err != nil {
		fmt.Println(err)
		return false
	}

	// glob matching
	parts := strings.Split(p, ".")
	for len(parts) > 0 {
		parts[len(parts)-1] = ""
		p = strings.Join(parts, ".")
		p += "*"

		c, err := au.C.Find(bson.M{
			"name":  u.Account,
			"perms": p,
		}).Count()

		if err != nil {
			fmt.Println(err)
			return false
		}
		if c > 0 {
			return true
		}

		parts = parts[:len(parts)-1]
	}

	return c > 0
}

func (au *GenericAuth) getHash() hash.Hash {
	h := md5.New()
	io.WriteString(h, au.Salt)
	return h
}

func (au *GenericAuth) LoginHandler(b *bot.Bot, m *irc.Message) {
	u := au.getUser(e.Prefix.Nick)
	if u.Account != "" {
		b.MentionReply(e, "you are already logged in as '%s'", u.Account)
		return
	}

	if len(u.Channels) == 0 {
		b.MentionReply(e, "You cannot log in if you're not in a channel with me")
		return
	}

	args := strings.SplitN(e.Trailing(), " ", 2)
	if len(args) != 2 {
		// TODO: Make internal help thing
		b.MentionReply(e, "wrong args")
		return
	}

	h := au.getHash()
	io.WriteString(h, args[1])

	pw := hex.EncodeToString(h.Sum(nil))

	cnt, err := au.C.Find(bson.M{
		"name":     args[0],
		"password": pw,
	}).Count()

	if err != nil {
		fmt.Println(err)
		return
	}

	if cnt > 0 {
		u.Account = args[0]
		b.MentionReply(e, "you are now logged in as '%s'", args[0])
		au.users[u.CurrentNick] = u
	} else {
		b.MentionReply(e, "login failed")
	}
}

func (au *GenericAuth) LogoutHandler(b *bot.Bot, e *irc.Event) {
	u := au.getUser(e.Identity.Nick)
	if u.Account == "" {
		b.MentionReply(e, "you are not logged in")
		return
	}

	u.Account = ""
	au.users[u.CurrentNick] = u
	b.MentionReply(e, "you have been logged out")
}

func (au *GenericAuth) RegisterHandler(b *bot.Bot, e *irc.Event) {
	u := au.getUser(e.Identity.Nick)
	if u.Account != "" {
		b.MentionReply(e, "you are already logged in as '%s'", u.Account)
		return
	}

	args := strings.SplitN(e.Trailing(), " ", 2)
	if len(args) < 2 {
		b.MentionReply(e, "wrong args")
		return
	}

	cnt, err := au.C.Find(bson.M{
		"name": args[0],
	}).Count()

	if err != nil {
		fmt.Println(err)
		return
	}

	if cnt > 0 {
		b.MentionReply(e, "there is already a user with that name")
		return
	}

	h := au.getHash()
	io.WriteString(h, args[1])

	err = au.C.Insert(bson.M{
		"name":     args[0],
		"password": hex.EncodeToString(h.Sum(nil)),
	})

	if err != nil {
		fmt.Println(err)
		return
	}

	u.Account = args[0]
	delete(au.users, e.Identity.Nick)
	au.users[e.Identity.Nick] = u

	b.MentionReply(e, "you have been registered and logged in")
}

func (au *GenericAuth) AddPermHandler(b *bot.Bot, e *irc.Event) {
	u := au.getUser(e.Identity.Nick)
	if u.Account == "" {
		b.MentionReply(e, "you are not logged in")
		return
	}

	if !au.userCan(u, "admin") && !au.userCan(u, "generic_auth.addperm") {
		b.MentionReply(e, "you don't have permission to add permissions")
		return
	}

	args := strings.Split(e.Trailing(), " ")
	if len(args) != 2 {
		b.MentionReply(e, "wrong args")
		return
	}

	a := genericAccount{}
	err := au.C.Find(bson.M{"name": args[0]}).One(&a)
	if err != nil {
		// NOTE: This may be another error?
		b.MentionReply(e, "account '%s' does not exist", args[0])
		return
	}

	if args[1] == "admin" && !au.userCan(u, "admin") {
		b.MentionReply(e, "only users with the 'admin' permission can add admins")
		return
	}

	for _, v := range a.Perms {
		if v == args[1] {
			b.MentionReply(e, "user '%s' already has perm '%s'", args[0], args[1])
			return
		}
	}

	au.C.UpdateId(a.Id, bson.M{"$push": bson.M{"perms": args[1]}})
	b.MentionReply(e, "added perm '%s' to user '%s'", args[1], args[0])
}

func (au *GenericAuth) DelPermHandler(b *bot.Bot, e *irc.Event) {
	u := au.getUser(e.Identity.Nick)
	if u.Account == "" {
		b.MentionReply(e, "you are not logged in")
		return
	}

	if !au.userCan(u, "admin") && !au.userCan(u, "generic_auth.delperm") {
		b.MentionReply(e, "you don't have permission to remove permissions")
		return
	}

	args := strings.Split(e.Trailing(), " ")
	if len(args) != 2 {
		b.MentionReply(e, "wrong args")
		return
	}

	if args[1] == "admin" && !au.userCan(u, "admin") {
		b.MentionReply(e, "only users with the 'admin' permission can remove admins")
		return
	}

	err := au.C.Update(bson.M{"name": args[0]}, bson.M{"$pull": bson.M{"perms": args[1]}})
	if err != nil {
		b.MentionReply(e, "account '%s' does not exist", args[0])
		return
	}

	b.MentionReply(e, "removed perm '%s' from user '%s'", args[1], args[0])
}

func (au *GenericAuth) CheckPermHandler(b *bot.Bot, e *irc.Event) {
	u := au.getUser(e.Identity.Nick)
	if u.Account == "" {
		b.MentionReply(e, "you are not logged in")
		return
	}

	if !au.userCan(u, "admin") && !au.userCan(u, "generic_auth.checkperms") {
		b.MentionReply(e, "you do not have permission to view permissions")
		return
	}

	args := strings.Split(e.Trailing(), " ")
	if len(args) != 1 {
		b.MentionReply(e, "wrong args")
		return
	}

	a := genericAccount{}
	err := au.C.Find(bson.M{"name": args[0]}).One(&a)
	if err != nil {
		b.MentionReply(e, "account '%s' does not exist", args[0])
		return
	}

	b.MentionReply(e, "permissions for '%s': %s", args[0], strings.Join(a.Perms, ", "))
}

func (au *GenericAuth) WhoisHandler(b *bot.Bot, e *irc.Event) {
	u := au.getUser(e.Identity.Nick)
	if u.Account == "" {
		b.MentionReply(e, "you are not logged in")
		return
	}

	if !au.userCan(u, "admin") && !au.userCan(u, "generic_auth.whois") {
		b.MentionReply(e, "you do not have permission to check a user account")
		return
	}

	args := strings.Split(e.Trailing(), " ")
	if len(args) != 1 || args[0] == "" {
		b.MentionReply(e, "wrong args")
		return
	}

	if cu, ok := au.users[args[0]]; ok && cu.Account != "" {
		b.MentionReply(e, "nick '%s' is user '%s'", args[0], cu.Account)
	} else {
		b.MentionReply(e, "nick '%s' is not logged in", args[0])
	}
}

func (au *GenericAuth) PasswdHandler(b *bot.Bot, e *irc.Event) {
	u := au.getUser(e.Identity.Nick)
	if u.Account == "" {
		b.MentionReply(e, "you are not logged in")
		return
	}

	args := strings.Split(e.Trailing(), " ")
	if len(args) != 1 || args[0] == "" {
		b.MentionReply(e, "wrong args")
		return
	}

	h := au.getHash()
	io.WriteString(h, args[0])

	_, err := au.C.Upsert(bson.M{"name": u.Account}, bson.M{"$set": bson.M{"password": hex.EncodeToString(h.Sum(nil))}})
	if err != nil {
		fmt.Println(err)
		return
	}

	b.MentionReply(e, "your password has been changed")
}

type GenericAuthConfig struct {
	Salt string
}

func (au *GenericAuth) Reload(b *bot.Bot) error {
	conf := &GenericAuthConfig{}
	err := b.LoadConfig("auth_generic", conf)
	if err != nil {
		return err
	}

	au.Salt = conf.Salt

	return nil
}

func NewGenericAuthPlugin(b *bot.Bot) (bot.AuthPlugin, error) {
	au := &GenericAuth{
		b.DB.C("generic_auth_accounts"),
		"",
		make(map[string]*user),
	}

	err := au.Reload(b)
	if err != nil {
		return nil, err
	}

	b.Event("001", au.connectHandler)
	b.Event("JOIN", au.joinHandler)
	b.Event("NICK", au.nickHandler)
	b.Event("PART", au.partHandler)
	b.Event("QUIT", au.quitHandler)
	b.Event("353", au.namreplyHandler)

	b.CommandPrivate("login", "[username] [password]", au.LoginHandler)
	b.CommandPrivate("logout", "", au.LogoutHandler)
	b.CommandPrivate("register", "[username] [password]", au.RegisterHandler)
	b.CommandPrivate("addperm", "[username] [perm]", au.AddPermHandler)
	b.CommandPrivate("delperm", "[username] [perm]", au.DelPermHandler)
	b.CommandPrivate("checkperm", "[username]", au.CheckPermHandler)
	b.CommandPrivate("whois", "[username]", au.WhoisHandler)
	b.CommandPrivate("passwd", "[password]", au.PasswdHandler)

	return au, nil
}

func (au *GenericAuth) CheckPerm(n string, p string) bool {
	u := au.getUser(n)
	return au.userCan(u, p) || au.userCan(u, "admin")
}

// user tracking utilities

func (au *GenericAuth) getUser(nick string) *user {
	u, ok := au.users[nick]
	if !ok {
		u = &user{CurrentNick: nick}
	}

	return u
}

func (au *GenericAuth) addChannelToNick(c, n string) {
	u := au.getUser(n)

	for i := 0; i < len(u.Channels); i++ {
		if u.Channels[i] == c {
			return
		}
	}

	u.Channels = append(u.Channels, c)
	au.users[n] = u
}

func (au *GenericAuth) removeChannelFromUser(c string, u *user) {
	for i := 0; i < len(u.Channels); i++ {
		if u.Channels[i] == c {
			// Swap with last element and shrink slice
			u.Channels[i] = u.Channels[len(u.Channels)-1]
			u.Channels = u.Channels[:len(u.Channels)-1]
			break
		}
	}

	if len(u.Channels) == 0 {
		// Removing user
		delete(au.users, u.CurrentNick)
	}
}

// user tracking

func (au *GenericAuth) connectHandler(b *bot.Bot, e *irc.Event) {
	au.users = make(map[string]*user)
}

func (au *GenericAuth) joinHandler(b *bot.Bot, e *irc.Event) {
	if e.Identity.Nick != b.C.CurrentNick() {
		au.addChannelToNick(e.Args[0], e.Identity.Nick)
	} else {
		for _, user := range au.users {
			au.removeChannelFromUser(e.Args[0], user)
		}
	}
}

func (au *GenericAuth) nickHandler(b *bot.Bot, e *irc.Event) {
	u := au.getUser(e.Identity.Nick)
	if len(u.Channels) == 0 {
		return
	}

	u.CurrentNick = e.Trailing()
	au.users[u.CurrentNick] = u

	u = au.getUser(u.CurrentNick)
	fmt.Println(u.CurrentNick)
}

func (au *GenericAuth) partHandler(b *bot.Bot, e *irc.Event) {
	if e.Identity.Nick != b.C.CurrentNick() {
		if u, ok := au.users[e.Identity.Nick]; ok {
			au.removeChannelFromUser(e.Args[0], u)
		}
	} else {
		for _, u := range au.users {
			au.removeChannelFromUser(e.Args[0], u)
		}
	}
}

func (au *GenericAuth) quitHandler(b *bot.Bot, e *irc.Event) {
	if e.Identity.Nick == b.C.CurrentNick() {
		// nop
		return
	}

	delete(au.users, e.Identity.Nick)
}

func isLetter(l byte) bool {
	return ((l >= 'a' && l <= 'z') || (l >= 'A' && l <= 'Z'))
}

func isSpecial(l byte) bool {
	specials := []byte("[]\\`_^{|}")
	return bytes.IndexByte(specials, l) != -1
}

func (au *GenericAuth) namreplyHandler(b *bot.Bot, e *irc.Event) {
	ch := e.Args[len(e.Args)-2]
	args := strings.Split(e.Trailing(), " ")

	for i := 0; i < len(args); i++ {
		n := args[i]
		for !isLetter(n[0]) && !isSpecial(n[0]) {
			n = n[1:]
		}

		au.addChannelToNick(ch, n)
	}
}
