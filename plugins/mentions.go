package plugins

import (
	"github.com/belak/irc"
	"github.com/belak/go-seabird/bot"
)

func init() {
	bot.RegisterPlugin("mentions", NewMentionsPlugin)
}

func NewMentionsPlugin(b *bot.Bot) (bot.Plugin, error) {
	b.MentionMux.Event(mentionsCallback)
	return nil, nil
}

func mentionsCallback(b *bot.Bot, m *irc.Message) {
	switch m.Trailing() {
	case "ping":
		b.MentionReply(m, "pong")
	case "scoobysnack", "scooby snack":
		b.Reply(m, "Scooby Dooby Doo!")
	case "botsnack", "bot snack":
		b.Reply(m, ":)")
	}
}
