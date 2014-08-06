package seabird

import (
	"strings"

	"bitbucket.org/belak/irc"
)

func SayHandler(c *irc.Client, e *irc.Event) {
	p := strings.SplitN(e.Trailing(), " ", 2)
	c.Writef("PRIVMSG %s :%s", p[0], p[1])
}
