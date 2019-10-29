package extra

import (
	seabird "github.com/belak/go-seabird"
)

func init() {
	seabird.RegisterPlugin("mentions", newMentionsPlugin)
}

func newMentionsPlugin(mm *seabird.MentionMux) {
	mm.Event(mentionsCallback)
}

func mentionsCallback(b *seabird.Bot, r *seabird.Request) {
	switch r.Message.Trailing() {
	case "ping":
		b.MentionReply(r, "pong")
	case "scoobysnack", "scooby snack":
		b.Reply(r, "Scooby Dooby Doo!")
	case "botsnack", "bot snack":
		b.Reply(r, ":)")
	}
}
