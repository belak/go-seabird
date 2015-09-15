package plugins

import (
	"github.com/belak/seabird/bot"
	"github.com/belak/irc"
)

func init() {
	bot.RegisterPlugin("mentions", NewMentionsPlugin)
}

type MentionsPlugin struct{}

func NewMentionsPlugin(b *bot.Bot) (bot.Plugin, error) {
	p := &MentionsPlugin{}
	b.MentionMux.Event(Mentions)
	return p, nil
}

func Mentions(b *bot.Bot, m *irc.Message) {
	switch m.Trailing() {
	case "ping":
		b.MentionReply(m, "pong")
	case "scoobysnack", "scooby snack":
		b.Reply(m, "Scooby Dooby Doo!")
	case "botsnack", "bot snack":
		b.Reply(m, ":)")
	}
}
