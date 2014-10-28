package seabird

import (
	"runtime"
	"time"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
)

func init() {
	bot.RegisterPlugin("ctcp", NewCTCPPlugin)
}

type CTCPPlugin struct{}

func NewCTCPPlugin(b *bot.Bot) (bot.Plugin, error) {
	p := &CTCPPlugin{}

	b.CTCP("TIME", p.Time)
	b.CTCP("PING", p.Ping)
	b.CTCP("VERSION", p.Version)

	return p, nil
}

func (p *CTCPPlugin) Time(b *bot.Bot, e *irc.Event) {
	t := time.Now().Format("Mon 2 Jan 2006 15:04:05 MST")
	b.CTCPReply(e, "TIME %s", t)
}

func (p *CTCPPlugin) Ping(b *bot.Bot, e *irc.Event) {
	b.CTCPReply(e, "PING %s", e.Trailing())
}

func (p *CTCPPlugin) Version(b *bot.Bot, e *irc.Event) {
	b.CTCPReply(e, "VERSION belak/seabird [%s %s %s]",
		runtime.GOOS, runtime.GOARCH, runtime.Version())
}
