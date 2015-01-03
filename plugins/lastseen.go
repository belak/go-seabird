package plugins

import (
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
)

func init() {
	bot.RegisterPlugin("lastseen", NewLastSeenPlugin)
}

type LastSeenPlugin struct {
	db *sqlx.DB
}

func NewLastSeenPlugin(c *mux.CommandMux, b *irc.BasicMux, db *sqlx.DB) error {
	p := &LastSeenPlugin{
		db,
	}

	c.Event("active", p.Active, &mux.HelpInfo{
		"<nick>",
		"Reports the last time user was seen",
	})
	b.Event("PRIVMSG", p.Msg)

	return nil
}

func (p *LastSeenPlugin) Active(c *irc.Client, e *irc.Event) {
	if e.Trailing() == "" {
		c.MentionReply(e, "Nick required")
		return
	}

	nick := e.Trailing()
	channel := e.Args[0]

	msg := p.getLastSeen(nick, channel)

	c.MentionReply(e, "%s", msg)
}

func (p *LastSeenPlugin) getLastSeen(nick, channel string) string {
	var lastseen int64
	err := p.db.Get(&lastseen, "SELECT lastseen FROM lastseen WHERE name=$1 AND channel=$2", p.CleanedName(nick), channel)
	if err != nil {
		return "Unknown user"
	}

	tm := time.Unix(lastseen, 0)

	if isActiveTime(lastseen) {
		return nick + " was last seen at " + formatTime(tm)
	} else {
		return nick + " was last seen on " + formatDate(tm) + " at " + formatTime(tm) + " (inactive)"
	}
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

func (p *LastSeenPlugin) isActive(nick, channel string) bool {
	var lastseen int64
	err := p.db.Get(&lastseen, "SELECT lastseen FROM lastseen WHERE name=$1 AND channel=$2", p.CleanedName(nick), channel)
	if err != nil {
		return false
	}

	return isActiveTime(lastseen)
}

func (p *LastSeenPlugin) Msg(c *irc.Client, e *irc.Event) {
	if len(e.Args) < 2 || !e.FromChannel() || e.Identity.Nick == "" {
		return
	}

	nick := e.Identity.Nick
	channel := e.Args[0]

	p.updateLastSeen(nick, channel)
}

func (p *LastSeenPlugin) CleanedName(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}

// Thanks to @belak for the comments
func (p *LastSeenPlugin) updateLastSeen(nick, channel string) {
	name := p.CleanedName(nick)
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
