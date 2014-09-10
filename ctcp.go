package seabird

import (
	"runtime"
	"strings"
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

	b.Event("CTCP", p.Time)
	b.Event("CTCP", p.Ping)
	b.Event("CTCP", p.Version)

	return p, nil
}

func (p *CtcpPlugin) Reload(b *bot.Bot) error {
	//noop
	return nil
}

func (p *CtcpPlugin) Time(b *bot.Bot, e *irc.Event) {
	if !strings.HasPrefix(e.Trailing(), "TIME") {
		return
	}

	t := time.Now().Format("Mon 2 Jan 2006 15:04:05 EST")
	b.CtcpReply(e, "TIME %s", t)
}

func (p *CtcpPlugin) Ping(b *bot.Bot, e *irc.Event) {
	if !strings.HasPrefix(e.Trailing(), "PING") {
		return
	}

	b.CtcpReply(e, e.Trailing())
}

func (p *CtcpPlugin) Version(b *bot.Bot, e *irc.Event) {
	if !strings.HasPrefix(e.Trailing(), "VERSION") {
		return
	}

	b.CtcpReply(e, "VERSION belak/seabird [%s %s]",
		runtime.GOOS, runtime.GOARCH)
}
