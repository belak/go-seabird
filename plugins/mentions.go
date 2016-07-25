package plugins

import (
	"github.com/belak/go-seabird/bot"
	"github.com/belak/irc"
)

func init() {
	bot.RegisterPlugin("mentions", newMentionsPlugin)
}

func newMentionsPlugin(b *bot.Bot) (bot.Plugin, error) {
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
