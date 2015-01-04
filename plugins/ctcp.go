package plugins

import (
	"bytes"
	"os/exec"
	"runtime"
	"time"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
)

func init() {
	bot.RegisterPlugin("ctcp", NewCTCPPlugin)
}

type CTCPPlugin struct {
	EnableGit bool
}

func NewCTCPPlugin(b *bot.Bot, m *mux.CTCPMux) error {
	p := &CTCPPlugin{}

	// NOTE: We ignore the error because ctcp config is optional
	b.Config("ctcp", p)

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
	if p.EnableGit {
		out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
		if err != nil {
			c.CTCPReply(e, "VERSION Error running git: %s", err)
			return
		}

		c.CTCPReply(e, "VERSION belak/seabird [%s %s %s] %s",
			runtime.GOOS, runtime.GOARCH, runtime.Version(), string(bytes.TrimSpace(out)))
	} else {
		c.CTCPReply(e, "VERSION belak/seabird [%s %s %s]",
			runtime.GOOS, runtime.GOARCH, runtime.Version())
	}
}
