package auth

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"bitbucket.org/belak/irc"
)

type GenericAccount struct {
	Id    bson.ObjectId `bson:"_id"`
	Name  string        `bson:"name"`
	Perms []string      `bson:"perms,omitempty"`
}

type User struct {
	CurrentNick string
	Account string
	Channels []string
}

type GenericAuth struct {
	Client *irc.Client
	C      *mgo.Collection
	Users  map[string]*User
	salt   string
}

func NewGenericAuth(c *irc.Client, db *mgo.Database, salt string) *GenericAuth{
	p := &GenericAuth{Client: c, C: db.C("generic_auth_accounts"), salt: salt}
	p.trackUsers()


	return p
}

func (au *GenericAuth) CheckPerm(p string, h irc.Handler) irc.Handler {
	return h
}

func (au *GenericAuth) CheckPermFunc(p string, f irc.HandlerFunc) irc.HandlerFunc{
	return f
}

// user tracking utilities

func (au *GenericAuth) GetUser(nick string) *User{
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
	au.Client.EventFunc("001",  au.connectHandler)
	au.Client.EventFunc("JOIN", au.joinHandler)
	au.Client.EventFunc("NICK", au.nickHandler)
	au.Client.EventFunc("PART", au.partHandler)
	au.Client.EventFunc("QUIT", au.quitHandler)
}

