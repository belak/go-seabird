package plugins

import (
	"strings"
	"time"

	"github.com/belak/go-seabird/seabird"
	"github.com/belak/irc"
	"github.com/belak/nut"
)

func init() {
	seabird.RegisterPlugin("remind", newreminderPlugin)
}

type reminderPlugin struct {
	db *nut.DB
}

type targetType int

const (
	channelTarget targetType = iota
	privateTarget
)

type reminder struct {
	Key          string
	Target       string
	TargetType   targetType
	Content      string
	ReminderTime time.Time
}

func newreminderPlugin(m *seabird.BasicMux, cm *seabird.CommandMux, db *nut.DB) error {
	p := &reminderPlugin{db: db}

	err := p.db.EnsureBucket("remind_reminders")
	if err != nil {
		return err
	}

	m.Event("001", p.InitialDispatch)
	m.Event("JOIN", p.JoinDispatch)

	cm.Event("remind", p.RemindCommand, &seabird.HelpInfo{
		Usage:       "<duration> <message>",
		Description: "Remind yourself to do something.",
	})

	return nil
}

func (p *reminderPlugin) dispatch(b *seabird.Bot, r *reminder) {
	logger := b.GetLogger().WithField("reminder", r)

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
	err := p.db.Update(func(tx *nut.Tx) error {
		bucket := tx.Bucket("remind_reminders")
		return bucket.Delete(r.Key)
	})

	if err != nil {
		logger.WithError(err).Error("Failed to remove reminder")
		return
	}

	logger.Info("Dispatched reminder")
}

func (p *reminderPlugin) dispatchReminders(b *seabird.Bot, tType targetType, target string) {
	logger := b.GetLogger()

	reminders := []*reminder{}

	err := p.db.View(func(tx *nut.Tx) error {
		bucket := tx.Bucket("remind_reminders")
		cursor := bucket.Cursor()

		v := &reminder{}

		for _, err := cursor.First(v); err == nil; _, err = cursor.Next(v) {
			if v.TargetType != tType {
				v = &reminder{}
				continue
			}

			if target == "" || v.Target != target {
				v = &reminder{}
				continue
			}

			reminders = append(reminders, v)

			v = &reminder{}
		}

		return nil
	})

	if err != nil && err != nut.ErrCursorEOF {
		logger.WithError(err).Error("Failed to look up private reminders for dispatch")
		return
	}

	for _, r := range reminders {
		go p.dispatch(b, r)
	}
}

// InitialDispatch is used to send private messages to users on connection. We
// can't queue up the channels yet because we haven't joined them.
func (p *reminderPlugin) InitialDispatch(b *seabird.Bot, m *irc.Message) {
	p.dispatchReminders(b, privateTarget, "")
}

// When we join a channel, we need to see if there are any reminders to be
// queued up.
func (p *reminderPlugin) JoinDispatch(b *seabird.Bot, m *irc.Message) {
	// If it's not the bot or we got an invalid message, we ignore it.
	if m.Prefix.Name != b.CurrentNick() || len(m.Params) < 1 {
		return
	}

	p.dispatchReminders(b, channelTarget, m.Params[0])
}

func (p *reminderPlugin) RemindCommand(b *seabird.Bot, m *irc.Message) {
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
		TargetType:   privateTarget,
		Content:      split[1],
		ReminderTime: time.Now().Add(dur),
	}

	if m.FromChannel() {
		// If it was from a channel, we need to prepend the user's name.
		r.Target = m.Params[0]
		r.TargetType = channelTarget
		r.Content = m.Prefix.Name + ": " + r.Content
	}

	err = p.db.Update(func(tx *nut.Tx) error {
		bucket := tx.Bucket("remind_reminders")

		key, err := bucket.NextID()
		if err != nil {
			return err
		}

		r.Key = key

		return bucket.Put(r.Key, r)
	})

	if err != nil {
		b.MentionReply(m, "Failed to store reminder: %s", err)
		return
	}

	b.MentionReply(m, "Event stored")

	go p.dispatch(b, r)
}
