package extra

import (
	"github.com/lrstanley/girc"

	seabird "github.com/belak/go-seabird"
)

func init() {
	seabird.RegisterPlugin("mentions", newMentionsPlugin)
}

func newMentionsPlugin(c *girc.Client) {
	c.Handlers.Add(seabird.MENTION, mentionsCallback)
}

func mentionsCallback(c *girc.Client, e girc.Event) {
	switch e.Last() {
	case "ping":
		c.Cmd.ReplyTof(e, "pong")
	case "scoobysnack", "scooby snack":
		c.Cmd.Replyf(e, "Scooby Dooby Doo!")
	case "botsnack", "bot snack":
		c.Cmd.Replyf(e, ":)")
	}
}
