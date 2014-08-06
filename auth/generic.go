package auth

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"strings"

	"bitbucket.org/belak/irc"
	"bitbucket.org/belak/irc/mux"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

type GenericAccount struct {
	Id    bson.ObjectId `bson:"_id"`
	Name  string        `bson:"name"`
	Perms []string      `bson:"perms,omitempty"`
}

type User struct {
	CurrentNick string
	Account     string
	Channels    []string
}

type GenericAuth struct {
	Client *irc.Client
	C      *mgo.Collection
	Users  map[string]*User
	Salt   string
}

func (au *GenericAuth) userCan(u *User, p string) bool {
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

	return c > 0
}

func (au *GenericAuth) getHash() hash.Hash {
	h := md5.New()
	io.WriteString(h, au.Salt)
	return h
}

func (au *GenericAuth) newLoginHandler(prefix string) irc.HandlerFunc {
	return func(c *irc.Client, e *irc.Event) {
		u := au.GetUser(e.Identity.Nick)
		if u.Account != "" {
			c.MentionReply(e, "you are already logged in as '%s'", u.Account)
			return
		}

		args := strings.SplitN(e.Trailing(), " ", 2)
		if len(args) != 2 {
			c.MentionReply(e, "usage: %slogin username password", prefix)
			return
		}

		h := au.getHash()
		io.WriteString(h, args[1])

		pw := hex.EncodeToString(h.Sum(nil))
		fmt.Printf("%s --- %s --- %s\n", au.Salt, args[1], pw)

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
			au.Client.MentionReply(e, "you are now logged in as '%s'", args[0])
			au.Users[u.CurrentNick] = u
		} else {
			au.Client.MentionReply(e, "login failed")
		}
	}
}

func (au *GenericAuth) newLogoutHandler(prefix string) irc.HandlerFunc {
	return func(c *irc.Client, e *irc.Event) {
		u := au.GetUser(e.Identity.Nick)
		if u.Account == "" {
			c.MentionReply(e, "you are not logged in")
			return
		}

		u.Account = ""
		au.Users[u.CurrentNick] = u
		c.MentionReply(e, "you have been logged out")
	}
}

func (au *GenericAuth) newRegisterHandler(prefix string) irc.HandlerFunc {
	return func(c *irc.Client, e *irc.Event) {
		u := au.GetUser(e.Identity.Nick)
		if u.Account != "" {
			c.MentionReply(e, "you are already logged in as '%s'", u.Account)
			return
		}

		args := strings.SplitN(e.Trailing(), " ", 2)
		if len(args) < 2 {
			c.MentionReply(e, "usage: %sregister <username> <password>", prefix)
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
			c.MentionReply(e, "there is already a user with that name")
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
		delete(au.Users, e.Identity.Nick)
		au.Users[e.Identity.Nick] = u

		c.MentionReply(e, "you have been registered and logged in")
	}
}

func (au *GenericAuth) newAddPermHandler(prefix string) irc.HandlerFunc {
	return func(c *irc.Client, e *irc.Event) {
		u := au.GetUser(e.Identity.Nick)
		if u.Account == "" {
			c.MentionReply(e, "you are not logged in")
			return
		}

		if !au.userCan(u, "admin") && !au.userCan(u, "generic_auth.addperm") {
			c.MentionReply(e, "you don't have permission to add permissions")
			return
		}

		args := strings.Split(e.Trailing(), " ")
		if len(args) != 2 {
			c.MentionReply(e, "usage: %saddperm <user> <perm>", prefix)
			return
		}

		a := GenericAccount{}
		err := au.C.Find(bson.M{"name": args[0]}).One(&a)
		if err != nil {
			// NOTE: This may be another error?
			c.MentionReply(e, "account '%s' does not exist", args[0])
			return
		}

		if args[1] == "admin" && !au.userCan(u, "admin") {
			c.MentionReply(e, "only users with the 'admin' permission can add admins")
			return
		}

		for _, v := range a.Perms {
			if v == args[1] {
				c.MentionReply(e, "user '%s' already has perm '%s'", args[0], args[1])
				return
			}
		}

		au.C.UpdateId(a.Id, bson.M{"$push": bson.M{"perms": args[1]}})
		c.MentionReply(e, "added perm '%s' to user '%s'", args[1], args[0])
	}
}

func (au *GenericAuth) newDelPermHandler(prefix string) irc.HandlerFunc {
	return func(c *irc.Client, e *irc.Event) {
		u := au.GetUser(e.Identity.Nick)
		if u.Account == "" {
			c.MentionReply(e, "you are not logged in")
			return
		}

		if !au.userCan(u, "admin") && !au.userCan(u, "generic_auth.delperm") {
			c.MentionReply(e, "you don't have permission to remove permissions")
			return
		}

		args := strings.Split(e.Trailing(), " ")
		if len(args) != 2 {
			c.MentionReply(e, "usage: %sdelperm <user> <perm>", prefix)
			return
		}

		if args[1] == "admin" && !au.userCan(u, "admin") {
			c.MentionReply(e, "only users with the 'admin' permission can remove admins")
			return
		}

		err := au.C.Update(bson.M{"name": args[0]}, bson.M{"$pull": bson.M{"perms": args[1]}})
		if err != nil {
			c.MentionReply(e, "account '%s' does not exist", args[0])
			return
		}

		c.MentionReply(e, "removed perm '%s' to user '%s'", args[1], args[0])
	}
}

func NewGenericAuth(c *irc.Client, db *mgo.Database, prefix string, salt string) *GenericAuth {
	au := &GenericAuth{Client: c, C: db.C("generic_auth_accounts"), Salt: salt}
	au.trackUsers()

	cmds := mux.NewCommandMux(prefix)
	cmds.PrivateFunc("login", au.newLoginHandler(prefix))
	cmds.PrivateFunc("logout", au.newLogoutHandler(prefix))
	cmds.PrivateFunc("register", au.newRegisterHandler(prefix))
	cmds.PrivateFunc("addperm", au.newAddPermHandler(prefix))
	cmds.PrivateFunc("delperm", au.newDelPermHandler(prefix))
	// TODO: !checkperms <user>
	c.Event("PRIVMSG", cmds)

	return au
}

func (au *GenericAuth) CheckPerm(p string, h irc.Handler) irc.Handler {
	return h
}

func (au *GenericAuth) CheckPermFunc(p string, f irc.HandlerFunc) irc.HandlerFunc {
	return func(c *irc.Client, e *irc.Event) {
		fmt.Println("xxxxx")
		u := au.GetUser(e.Identity.Nick)
		if au.userCan(u, p) {
			f(c, e)
		} else {
			c.MentionReply(e, "You do not have the required permission: %s", p)
		}
	}
}

// user tracking utilities

func (au *GenericAuth) GetUser(nick string) *User {
	u, ok := au.Users[nick]
	if !ok {
		u = &User{CurrentNick: nick}
	}

	return u
}

func (au *GenericAuth) addChannelToNick(c, n string) {
	u := au.GetUser(n)

	for i := 0; i < len(u.Channels); i++ {
		if u.Channels[i] == c {
			return
		}
	}

	u.Channels = append(u.Channels, c)
	au.Users[n] = u
}

func (au *GenericAuth) removeChannelFromUser(c string, u *User) {
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
		delete(au.Users, u.CurrentNick)
	}
}

// user tracking

func (au *GenericAuth) connectHandler(c *irc.Client, e *irc.Event) {
	au.Users = make(map[string]*User)
}

func (au *GenericAuth) joinHandler(c *irc.Client, e *irc.Event) {
	if e.Identity.Nick != c.CurrentNick() {
		au.addChannelToNick(e.Args[0], e.Identity.Nick)
	} else {
		for _, user := range au.Users {
			au.removeChannelFromUser(e.Args[0], user)
		}
	}
}

func (au *GenericAuth) nickHandler(c *irc.Client, e *irc.Event) {
	u := au.GetUser(e.Identity.Nick)
	if len(u.Channels) == 0 {
		return
	}

	u.CurrentNick = e.Args[1]
	delete(au.Users, e.Identity.Nick)
	au.Users[u.CurrentNick] = u
}

func (au *GenericAuth) partHandler(c *irc.Client, e *irc.Event) {
	if e.Identity.Nick != c.CurrentNick() {
		if u, ok := au.Users[e.Identity.Nick]; ok {
			au.removeChannelFromUser(e.Args[0], u)
		}
	} else {
		for _, u := range au.Users {
			au.removeChannelFromUser(e.Args[0], u)
		}
	}
}

func (au *GenericAuth) quitHandler(c *irc.Client, e *irc.Event) {
	// TODO implement this
}

func (au *GenericAuth) trackUsers() {
	au.Client.EventFunc("001", au.connectHandler)
	au.Client.EventFunc("JOIN", au.joinHandler)
	au.Client.EventFunc("NICK", au.nickHandler)
	au.Client.EventFunc("PART", au.partHandler)
	au.Client.EventFunc("QUIT", au.quitHandler)
}
