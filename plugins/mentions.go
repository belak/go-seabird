package plugins

import (
	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
)

func init() {
	bot.RegisterPlugin("mentions", NewMentionsPlugin)
}

type MentionsPlugin struct{}

func NewMentionsPlugin(m *mux.MentionMux) error {
	m.Event(Mentions)
	return nil
}

func Mentions(c *irc.Client, e *irc.Event) {
	switch e.Trailing() {
	case "ping":
		c.MentionReply(e, "pong")
	case "scoobysnack", "scooby snack":
		c.Reply(e, "Scooby Dooby Doo!")
	case "botsnack", "bot snack":
		c.Reply(e, ":)")
	}
}
