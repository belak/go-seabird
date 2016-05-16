package bot

import (
	"github.com/belak/irc"
)

// AuthUser is the user returned by an AuthProvider's LookupUser method.
type AuthUser interface {
	HasPerm(b *Bot, perm string) bool
}

// AuthProvider is a special type of plugin which allows for perms to be looked
// up so plugins can limit access. The default implementation always returns a
// user object and always returns true for HasPerm.
type AuthProvider interface {
	LookupUser(b *Bot, id *irc.Prefix) AuthUser
}

// Everything below this point is for the default auth implementation
type nullAuthUser struct{}

func (u *nullAuthUser) HasPerm(b *Bot, perm string) bool {
	return true
}

type nullAuthProvider struct{}

func (a *nullAuthProvider) LookupUser(b *Bot, id *irc.Prefix) AuthUser {
	return &nullAuthUser{}
}
