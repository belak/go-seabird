package seabird

import "bitbucket.org/belak/irc"

func MentionsHandler(c *irc.Client, e *irc.Event) {
	switch e.Trailing() {
	case "ping":
		c.MentionReply(e, "pong")
	case "scoobysnack", "scooby snack":
		c.Reply(e, "Scooby Dooby Doo!")
	case "botsnack", "bot snack":
		c.Reply(e, ":)")
	}
}
