// +build ignore

package plugins

import (
	"bytes"
	"os/exec"
	"runtime"
	"time"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
)

type CTCPPlugin struct {
	EnableGit bool
}

func NewCTCPPlugin() bot.Plugin {
	return &CTCPPlugin{}
}

func (p *CTCPPlugin) Register(b *bot.Bot) error {
	// NOTE: We ignore the error because ctcp config is optional
	b.Config("ctcp", p)

	b.CTCPMux.Event("TIME", p.Time)
	b.CTCPMux.Event("PING", p.Ping)
	b.CTCPMux.Event("VERSION", p.Version)

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
