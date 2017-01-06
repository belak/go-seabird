package extra

import (
	"github.com/belak/go-seabird"
	"github.com/go-irc/irc"
)

func init() {
	seabird.RegisterPlugin("mentions", newMentionsPlugin)
}

func newMentionsPlugin(mm *seabird.MentionMux) {
	mm.Event(mentionsCallback)
}

func mentionsCallback(b *seabird.Bot, m *irc.Message) {
	switch m.Trailing() {
	case "ping":
		b.MentionReply(m, "pong")
	case "scoobysnack", "scooby snack":
		b.Reply(m, "Scooby Dooby Doo!")
	case "botsnack", "bot snack":
		b.Reply(m, ":)")
	}
}
