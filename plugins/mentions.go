package plugins

import (
	"encoding/json"

	seabird ".."
	"github.com/thoj/go-ircevent"
)

func init() {
	seabird.RegisterPlugin("mentions", NewMentionsPlugin)
}

type MentionsPlugin struct {
	Bot *seabird.Bot
}

func NewMentionsPlugin(b *seabird.Bot, d json.RawMessage) {
	p := &MentionsPlugin{b}
	b.RegisterMention(p.SnackCallback)
}

func (p *MentionsPlugin) SnackCallback(e *irc.Event) {
	switch e.Message() {
	case "scoobysnack", "scooby snack":
		p.Bot.Reply(e, "Scooby Dooby Doo!")
	case "botsnack", "bot snack":
		p.Bot.Reply(e, ":)")
	}
}
