package extra

import (
	"time"

	seabird "github.com/belak/go-seabird"
)

type uptimePlugin struct {
	startTime time.Time
}

func init() {
	seabird.RegisterPlugin("uptime", newUptimePlugin)
}

func newUptimePlugin(b *seabird.Bot) error {
	if err := b.EnsurePlugin("db"); err != nil {
		return err
	}

	p := &uptimePlugin{
		startTime: time.Now(),
	}

	cm := b.CommandMux()

	cm.Event("uptime", p.uptimeCallback, &seabird.HelpInfo{
		Description: "Display how long the bot has been running",
	})

	return nil
}

func (p *uptimePlugin) uptimeCallback(r *seabird.Request) {
	r.MentionReply("I have been running for %s", time.Since(p.startTime).Truncate(time.Second))
}
