package seabird

import "bitbucket.org/belak/irc"

func JoinHandler(c *irc.Client, e *irc.Event) {
	ch := e.Trailing()
	if ch == ""  {
		c.MentionReply(e, "usage: !join <channel>")
		return
	}

	c.Writef("JOIN %s", ch)
}

func PartHandler(c *irc.Client, e *irc.Event) {
	msg := e.Trailing()
	if msg == "" {
		msg = "I guess I'm not wanted here"
	}

	c.Writef("PART %s :%s", e.Args[0], msg)
}
