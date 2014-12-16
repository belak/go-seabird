package plugins

import (
	"runtime"
	"time"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
)

func init() {
	bot.RegisterPlugin("ctcp", NewCTCPPlugin)
}

type CTCPPlugin struct{}

func NewCTCPPlugin(m *mux.CTCPMux) error {
	p := &CTCPPlugin{}

	m.Event("TIME", p.Time)
	m.Event("PING", p.Ping)
	m.Event("VERSION", p.Version)

	return nil
}

func (p *CTCPPlugin) Time(c *irc.Client, e *irc.Event) {
	t := time.Now().Format("Mon 2 Jan 2006 15:04:05 MST")
	c.CTCPReply(e, "TIME %s", t)
}

func (p *CTCPPlugin) Ping(c *irc.Client, e *irc.Event) {
	c.CTCPReply(e, "PING %s", e.Trailing())
}

func (p *CTCPPlugin) Version(c *irc.Client, e *irc.Event) {
	c.CTCPReply(e, "VERSION belak/seabird [%s %s %s]",
		runtime.GOOS, runtime.GOARCH, runtime.Version())
}
