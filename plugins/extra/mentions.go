package extra

import (
	seabird "github.com/belak/go-seabird"
)

func init() {
	seabird.RegisterPlugin("mentions", newMentionsPlugin)
}

func newMentionsPlugin(b *seabird.Bot) error {
	mm := b.MentionMux()

	mm.Event(mentionsCallback)

	return nil
}

func mentionsCallback(r *seabird.Request) {
	switch r.Message.Trailing() {
	case "ping":
		r.MentionReply("pong")
	case "scoobysnack", "scooby snack":
		r.Reply("Scooby Dooby Doo!")
	case "botsnack", "bot snack":
		r.Reply(":)")
	case "pizzahousesnack":
		r.Reply("HECK YEAHHHHHHHHHHHH OMG I LOVE U THE WORLD IS GREAT")
	}
}
