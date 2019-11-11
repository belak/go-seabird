package extra

import (
	"time"

	"github.com/go-xorm/xorm"

	seabird "github.com/belak/go-seabird"
)

func init() {
	seabird.RegisterPlugin("watchdog", newWatchdogPlugin)
}

type watchdogPlugin struct {
	db *xorm.Engine
}

type watchdogCheck struct {
	Time   time.Time `xorm:"created"`
	Entity string
	Nonce  string
}

func newWatchdogPlugin(b *seabird.Bot) error {
	if err := b.EnsurePlugin("db"); err != nil {
		return err
	}

	p := &watchdogPlugin{
		db: CtxDB(b.Context()),
	}

	// Migrate any relevant tables
	err := p.db.Sync(watchdogCheck{})
	if err != nil {
		return err
	}

	cm := b.CommandMux()

	cm.Event("watchdog-check", p.check, &seabird.HelpInfo{
		Description: "Used to check availability of Seabird optionally including its DB",
	})

	return nil
}

func (p *watchdogPlugin) checkDb(r *seabird.Request, nonce string) bool {
	check := &watchdogCheck{
		Entity: r.Message.Prefix.String(),
		Nonce:  nonce,
	}

	_, err := p.db.Transaction(func(s *xorm.Session) (interface{}, error) {
		return s.Insert(check)
	})

	if err != nil {
		r.MentionReply("Error writing check to DB: \"%s\"", err)
		return false
	}

	return true
}

func (p *watchdogPlugin) check(r *seabird.Request) {
	timer := r.Timer("watchdog-check")
	defer timer.Done()

	if len(r.Message.Trailing()) == 0 {
		r.MentionReply("Error: missing nonce argument")
		return
	}

	nonce := r.Message.Trailing()

	if ok := p.checkDb(r, nonce); !ok {
		return
	}

	r.MentionReply("%d %s", time.Now().Unix(), nonce)
}
