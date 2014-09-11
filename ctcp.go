package seabird

import (
	"runtime"
	"time"

	"bitbucket.org/belak/irc"
	"bitbucket.org/belak/seabird/bot"
)

func init() {
	bot.RegisterPlugin("ctcp", NewCtcpPlugin)
}

type CtcpPlugin struct {}

func NewCtcpPlugin(b *bot.Bot) (bot.Plugin, error) {
	p := &CtcpPlugin{}

	b.Ctcp("TIME", p.Time)
	b.Ctcp("PING", p.Ping)
	b.Ctcp("VERSION", p.Version)

	return p, nil
}

func (p *CtcpPlugin) Reload(b *bot.Bot) error {
	//noop
	return nil
}

func (p *CtcpPlugin) Time(b *bot.Bot, e *irc.Event) {
	t := time.Now().Format("Mon 2 Jan 2006 15:04:05 EST")
	b.CtcpReply(e, "TIME %s", t)
}

func (p *CtcpPlugin) Ping(b *bot.Bot, e *irc.Event) {
	b.CtcpReply(e, e.Trailing())
}

func (p *CtcpPlugin) Version(b *bot.Bot, e *irc.Event) {
	b.CtcpReply(e, "VERSION belak/seabird [%s %s]",
		runtime.GOOS, runtime.GOARCH)
}
