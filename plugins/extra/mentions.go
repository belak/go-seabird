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
		r.MentionReplyf("pong")
	case "scoobysnack", "scooby snack":
		r.Replyf("Scooby Dooby Doo!")
	case "botsnack", "bot snack":
		r.Replyf(":)")
	case "pizzahousesnack":
		r.Replyf("HECK YEAHHHHHHHHHHHH OMG I LOVE U THE WORLD IS GREAT")
	}
}
