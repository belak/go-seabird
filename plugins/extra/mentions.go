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
		r.MentionReply("pong")
	case "scoobysnack", "scooby snack":
		r.Reply("Scooby Dooby Doo!")
	case "botsnack", "bot snack":
		r.Reply(":)")
	}
}
