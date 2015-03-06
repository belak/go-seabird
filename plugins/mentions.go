package plugins

import (
	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
)

type MentionsPlugin struct{}

func NewMentionsPlugin() bot.Plugin {
	return &MentionsPlugin{}
}

func (p *MentionsPlugin) Register(b *bot.Bot) error {
	b.MentionMux.Event(Mentions)
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
