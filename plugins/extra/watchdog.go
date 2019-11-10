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

func newWatchdogPlugin(b *seabird.Bot, m *seabird.BasicMux, cm *seabird.CommandMux, db *xorm.Engine) error {
	p := &watchdogPlugin{db: db}

	// Migrate any relevant tables
	err := db.Sync(watchdogCheck{})
	if err != nil {
		return err
	}

	cm.Event("watchdog-check", p.check, &seabird.HelpInfo{
		Description: "Used to check availability of Seabird optionally including its DB",
	})

	return nil
}

func (p *watchdogPlugin) checkDb(b *seabird.Bot, r *seabird.Request, nonce string) bool {
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

func (p *watchdogPlugin) check(b *seabird.Bot, r *seabird.Request) {
	timer := r.Timer("watchdog-check")
	defer timer.Done()

	if len(r.Message.Trailing()) == 0 {
		r.MentionReply("Error: missing nonce argument")
		return
	}

	nonce := r.Message.Trailing()

	ok := p.checkDb(b, r, nonce)
	if !ok {
		return
	}

	r.MentionReply("%d %s", time.Now().Unix(), nonce)
}
