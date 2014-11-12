package plugins

import (
	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
)

func init() {
	bot.RegisterPlugin("mentions", NewMentionsPlugin)
}

type MentionsPlugin struct{}

func NewMentionsPlugin(b *bot.Bot) (bot.Plugin, error) {
	p := &MentionsPlugin{}
	b.Mention(p.Mentions)
	return p, nil
}

func (p *MentionsPlugin) Mentions(b *bot.Bot, e *irc.Event) {
	switch e.Trailing() {
	case "ping":
		b.MentionReply(e, "pong")
	case "scoobysnack", "scooby snack":
		b.Reply(e, "Scooby Dooby Doo!")
	case "botsnack", "bot snack":
		b.Reply(e, ":)")
	}
}
