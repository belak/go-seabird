package seabird

import (
	"strings"

	"bitbucket.org/belak/irc"
)

func SayHandler(c *irc.Client, e *irc.Event) {
	p := strings.SplitN(e.Trailing(), " ", 2)
	if len(p) != 2 {
		c.MentionReply(e, "usage: !say <channel> <msg>")
		return
	}

	c.Writef("PRIVMSG %s :%s", p[0], p[1])
}
