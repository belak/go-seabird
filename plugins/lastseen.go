package plugins

import (
	"fmt"
	"strings"
	"time"

	"github.com/belak/go-seabird/seabird"
	"github.com/belak/irc"
	"github.com/jmoiron/sqlx"
)

func init() {
	seabird.RegisterPlugin("lastseen", newLastSeenPlugin)
}

type lastSeenPlugin struct {
	db *sqlx.DB
}

func newLastSeenPlugin(m *seabird.BasicMux, cm *seabird.CommandMux, db *sqlx.DB) {
	p := &lastSeenPlugin{db: db}

	cm.Event("active", p.activeCallback, &seabird.HelpInfo{
		Usage:       "<nick>",
		Description: "Reports the last time user was seen",
	})

	m.Event("PRIVMSG", p.msgCallback)
}

func (p *lastSeenPlugin) activeCallback(b *seabird.Bot, m *irc.Message) {
	nick := m.Trailing()
	if nick == "" {
		b.MentionReply(m, "Nick required")
		return
	}

	channel := m.Params[0]
	msg := p.getLastSeen(nick, channel)

	b.MentionReply(m, "%s", msg)
}

func (p *lastSeenPlugin) getLastSeen(nick, channel string) string {
	var lastseen int64
	err := p.db.Get(&lastseen, "SELECT lastseen FROM lastseen WHERE name=$1 AND channel=$2", strings.ToLower(nick), channel)
	if err != nil {
		return "Unknown user"
	}

	tm := time.Unix(lastseen, 0)

	if isActiveTime(lastseen) {
		return nick + " was last seen at " + formatTime(tm)
	}

	return nick + " was last seen on " + formatDate(tm) + " at " + formatTime(tm) + " (inactive)"
}

func isActiveTime(lastseen int64) bool {
	tm := time.Unix(lastseen, 0)
	now := time.Now()
	now = now.Add(-5 * time.Minute)
	return tm.After(now) || tm.Equal(now)
}

func formatTime(t time.Time) string {
	return fmt.Sprintf("%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
}

func formatDate(t time.Time) string {
	return fmt.Sprintf("%d %s %d", t.Day(), t.Month().String(), t.Year())
}

func (p *lastSeenPlugin) isActive(nick, channel string) bool {
	var lastseen int64
	err := p.db.Get(&lastseen, "SELECT lastseen FROM lastseen WHERE name=$1 AND channel=$2", strings.ToLower(nick), channel)
	if err != nil {
		return false
	}

	return isActiveTime(lastseen)
}

func (p *lastSeenPlugin) msgCallback(b *seabird.Bot, m *irc.Message) {
	if len(m.Params) < 2 || !m.FromChannel() || m.Prefix.Name == "" {
		return
	}

	nick := m.Prefix.Name
	channel := m.Params[0]

	p.updateLastSeen(nick, channel)
}

// Thanks to @belak for the comments
func (p *lastSeenPlugin) updateLastSeen(nick, channel string) {
	name := strings.ToLower(nick)
	now := time.Now().Unix()

	_, err := p.db.Exec("INSERT INTO lastseen VALUES ($1, $2, $3)", name, channel, now)
	// If it was a nil error, we got the insert
	if err == nil {
		return
	}

	// Grab a transaction, just in case
	tx, err := p.db.Beginx()
	defer tx.Commit()

	if err != nil {
		return
	}

	// If there was an error, we try an update.
	tx.Exec("UPDATE lastseen SET lastseen=$1 WHERE name=$2 AND channel=$3", now, name, channel)
}
