package plugins

import (
	"github.com/belak/seabird/bot"
	"github.com/belak/sorcix-irc"
)

type MentionsPlugin struct{}

func NewMentionsPlugin() bot.Plugin {
	return &MentionsPlugin{}
}

func (p *MentionsPlugin) Register(b *bot.Bot) error {
	b.MentionMux.Event(Mentions)
	return nil
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
