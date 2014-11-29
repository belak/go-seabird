// +build ignore

package plugins

import (
	"strings"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
)

func init() {
	bot.RegisterPlugin("chanops", NewChanOpsPlugin)
}

type ChanOpsPlugin struct{}

func NewChanOpsPlugin(b *bot.Bot) (bot.Plugin, error) {
	p := &ChanOpsPlugin{}
	b.Command("join", "[channel]", p.Join)
	b.Command("part", "", p.Part)
	b.Command("say", "[dest] [message]", p.Say)
	return p, nil
}

func (p *ChanOpsPlugin) Join(b *bot.Bot, e *irc.Event) {
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

func (p *ChanOpsPlugin) Part(b *bot.Bot, e *irc.Event) {
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

func (p *ChanOpsPlugin) Say(b *bot.Bot, e *irc.Event) {
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
