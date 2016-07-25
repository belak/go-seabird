package plugins

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/belak/go-seabird/bot"
	"github.com/belak/irc"
	"github.com/jmoiron/sqlx"
)

func init() {
	bot.RegisterPlugin("remind", newreminderPlugin)
}

type reminderPlugin struct {
	db *sqlx.DB
}

type reminder struct {
	ID           int64
	Target       string
	TargetType   string `db:"target_type"`
	Content      string
	ReminderTime time.Time `db:"reminder_time"`
}

func newreminderPlugin(b *bot.Bot) (bot.Plugin, error) {
	b.LoadPlugin("db")
	p := &reminderPlugin{b.Plugins["db"].(*sqlx.DB)}

	b.BasicMux.Event("001", p.InitialDispatch)
	b.BasicMux.Event("JOIN", p.JoinDispatch)
	b.CommandMux.Event("remind", p.RemindCommand, &bot.HelpInfo{
		Usage:       "<duration> <message>",
		Description: "Remind yourself to do something.",
	})

	return nil, nil
}

func (p *reminderPlugin) dispatch(b *bot.Bot, r *reminder) {
	// Because time.Sleep handles negative values (and 0) by simply
	// returning, this will be handled correctly even with negative
	// durations.
	waitDur := r.ReminderTime.Sub(time.Now())

	// Try to sleep this goroutine until the message needs to be delivered
	time.Sleep(waitDur)

	// Send the message
	b.Send(&irc.Message{
		Prefix:  &irc.Prefix{},
		Command: "PRIVMSG",
		Params:  []string{r.Target, r.Content},
	})

	// Nuke the reminder now that it's been sent
	_, err := p.db.Exec("DELETE FROM reminders WHERE id=$1", r.ID)
	if err != nil {
		fmt.Println(err)
		return
	}
}

// InitialDispatch is used to send private messages to users on connection. We
// can't queue up the channels yet because we haven't joined them.
func (p *reminderPlugin) InitialDispatch(b *bot.Bot, m *irc.Message) {
	reminders := []*reminder{}
	err := p.db.Select(&reminders, "SELECT * FROM reminders WHERE target_type=$1", "private")
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, r := range reminders {
		go p.dispatch(b, r)
	}
}

// When we join a channel, we need to see if there are any reminders to be
// queued up.
func (p *reminderPlugin) JoinDispatch(b *bot.Bot, m *irc.Message) {
	// If it's not the bot, we ignore it.
	if m.Prefix.Name != b.CurrentNick() || len(m.Params) < 1 {
		return
	}

	reminders := []*reminder{}
	err := p.db.Select(&reminders, "SELECT * FROM reminders WHERE target_type=$1 AND target=$2", "public", m.Params[0])
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, r := range reminders {
		go p.dispatch(b, r)
	}
}

func (p *reminderPlugin) RemindCommand(b *bot.Bot, m *irc.Message) {
	split := strings.SplitN(m.Trailing(), " ", 2)
	if len(split) != 2 {
		b.MentionReply(m, "Not enough args")
		return
	}

	dur, err := time.ParseDuration(split[0])
	if err != nil {
		b.MentionReply(m, "Invalid duration: %s", err)
		return
	}

	r := &reminder{
		Target:       m.Prefix.Name,
		TargetType:   "private",
		Content:      split[1],
		ReminderTime: time.Now().Add(dur),
	}

	if m.FromChannel() {
		// If it was from a channel, we need to prepend the user's name.
		r.Target = m.Params[0]
		r.TargetType = "public"
		r.Content = m.Prefix.Name + ": " + r.Content
	}

	// pq doesn't support sql.Result.LastInsertId so we hack around it by
	// doing this. Don't do this at home, kids!
	if p.db.DriverName() == "postgres" {
		var rowID int64
		err = p.db.QueryRow(
			"INSERT INTO reminders (target, target_type, content, reminder_time) VALUES ($1, $2, $3, $4) RETURNING id",
			r.Target, r.TargetType, r.Content, r.ReminderTime).Scan(&rowID)
		r.ID = rowID
	} else {
		var result sql.Result
		result, err = p.db.Exec(
			"INSERT INTO reminders (target, target_type, content, reminder_time) VALUES ($1, $2, $3, $4)",
			r.Target, r.TargetType, r.Content, r.ReminderTime)

		if err == nil {
			r.ID, err = result.LastInsertId()
		}
	}

	if err != nil {
		b.MentionReply(m, "Failed to store reminder: %s", err)
		return
	}

	b.MentionReply(m, "Event stored")

	go p.dispatch(b, r)
}
