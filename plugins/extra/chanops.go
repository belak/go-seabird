// +build ignore

package seabird_extra_plugins

import (
	"strings"

	"github.com/belak/irc"
	"github.com/belak/go-seabird/bot"
)

func init() {
	bot.RegisterPlugin("chanops", NewChanOpsPlugin)
}

func NewChanOpsPlugin(b *bot.Bot) (bot.Plugin, error) {
	b.Command("join", "[channel]", chanopsJoin)
	b.Command("part", "", chanopsPart)
	b.Command("say", "[dest] [message]", chanopsSay)
	return nil, nil
}

func chanopsJoin(b *bot.Bot, e *irc.Event) {
	if !b.CheckPerm(e.Identity.Nick, "chanops.join") {
		b.MentionReply(e, "You don't have permission to do that!")
		return
	}

	ch := e.Trailing()
	if ch == "" {
		b.MentionReply(e, "usage: !join <channel>")
		return
	}

	b.C.Writef("JOIN %s", ch)
}

func chanopsPart(b *bot.Bot, e *irc.Event) {
	if !b.CheckPerm(e.Identity.Nick, "chanops.part") {
		b.MentionReply(e, "You don't have permission to do that!")
		return
	}

	msg := e.Trailing()
	if msg == "" {
		msg = "I guess I'm not wanted here"
	}

	b.C.Writef("PART %s :%s", e.Args[0], msg)
}

func chanopsSay(b *bot.Bot, e *irc.Event) {
	if !b.CheckPerm(e.Identity.Nick, "chanops.say") {
		b.MentionReply(e, "You don't have permission to do that!")
		return
	}

	m := strings.SplitN(e.Trailing(), " ", 2)
	if len(m) != 2 {
		b.MentionReply(e, "usage: !say <channel> <msg>")
		return
	}

	b.C.Writef("PRIVMSG %s :%s", m[0], m[1])
}
