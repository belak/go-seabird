package extra

import (
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/belak/go-seabird"
	"github.com/belak/nut"
	irc "github.com/go-irc/irc/v2"
)

func init() {
	seabird.RegisterPlugin("remind", newreminderPlugin)
}

var timeRegexp = regexp.MustCompile(`\d+[smhd]`)

type reminderPlugin struct {
	db *nut.DB

	roomLock *sync.Mutex
	rooms    map[string]bool

	// Singly buffered channel
	updateChan chan struct{}
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
	p := &reminderPlugin{
		db:         db,
		roomLock:   &sync.Mutex{},
		rooms:      make(map[string]bool),
		updateChan: make(chan struct{}, 1),
	}

	err := p.db.EnsureBucket("remind_reminders")
	if err != nil {
		return err
	}

	m.Event("001", p.InitialDispatch)
	m.Event("JOIN", p.joinHandler)
	m.Event("PART", p.partHandler)
	m.Event("KICK", p.kickHandler)

	cm.Event("remind", p.RemindCommand, &seabird.HelpInfo{
		Usage:       "<duration> <message>",
		Description: "Remind yourself to do something.",
	})

	return nil
}

func (p *reminderPlugin) joinHandler(b *seabird.Bot, m *irc.Message) {
	if m.Prefix.Name != b.CurrentNick() {
		return
	}

	p.roomLock.Lock()
	defer p.roomLock.Unlock()
	p.rooms[m.Params[0]] = true

	p.updateChan <- struct{}{}
}

func (p *reminderPlugin) partHandler(b *seabird.Bot, m *irc.Message) {
	if m.Prefix.Name != b.CurrentNick() {
		return
	}

	p.roomLock.Lock()
	defer p.roomLock.Unlock()
	delete(p.rooms, m.Params[0])

	p.updateChan <- struct{}{}
}

func (p *reminderPlugin) kickHandler(b *seabird.Bot, m *irc.Message) {
	if m.Params[1] != b.CurrentNick() {
		return
	}

	p.roomLock.Lock()
	defer p.roomLock.Unlock()
	delete(p.rooms, m.Params[0])

	p.updateChan <- struct{}{}
}

func (p *reminderPlugin) nextReminder() (*reminder, error) {
	// Find the next reminder we'll have to send
	var r *reminder

	err := p.db.View(func(tx *nut.Tx) error {
		// Grab the room lock for this transaction
		p.roomLock.Lock()
		defer p.roomLock.Unlock()

		bucket := tx.Bucket("remind_reminders")
		cursor := bucket.Cursor()

		v := &reminder{}
		for _, err := cursor.First(v); err == nil; _, err = cursor.Next(v) {
			// If it's a channel target and we're not in the room,
			// we need to skip it
			if v.TargetType == channelTarget && !p.rooms[v.Target] {
				continue
			}

			// If we don't currently have a reminder or the new
			// reminder should be sent before our current one, we
			// update it.
			if r == nil || v.ReminderTime.Before(r.ReminderTime) {
				// Make absolutely sure that we have a copy of the
				// data because as soon as we call Next, it will go
				// away.
				tmp := *v
				r = &tmp
			}
		}

		return nil
	})

	return r, err
}

func (p *reminderPlugin) remindLoop(b *seabird.Bot) {
	logger := b.GetLogger()

	for {
		r, err := p.nextReminder()
		if err != nil {
			logger.WithError(err).Error("Transaction failure. Exiting loop.")
			return
		}

		var timer <-chan time.Time
		if r != nil {
			logger.WithField("reminder", r).Debug("Next reminder")

			waitDur := r.ReminderTime.Sub(time.Now())
			if waitDur <= 0 {
				p.dispatch(b, r)
				continue
			}

			timer = time.After(waitDur)
		}

		select {
		case <-timer:
			p.dispatch(b, r)
		case <-p.updateChan:
			continue
		}
	}
}

func (p *reminderPlugin) dispatch(b *seabird.Bot, r *reminder) {
	logger := b.GetLogger().WithField("reminder", r)

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
	}

	logger.Debug("Dispatched reminder")
}

// InitialDispatch is used to send private messages to users on connection. We
// can't queue up the channels yet because we haven't joined them.
func (p *reminderPlugin) InitialDispatch(b *seabird.Bot, m *irc.Message) {
	go p.remindLoop(b)
}

// ParseTime parses the text string and turns it into a time.Duration
func (p *reminderPlugin) ParseTime(timeStr string) (time.Duration, error) {
	var ret time.Duration

	for _, match := range timeRegexp.FindAllString(timeStr, -1) {
		switch match[len(match)-1] {
		case 's', 'm', 'h':
			tmp, err := time.ParseDuration(match)
			if err != nil {
				return ret, err
			}
			ret += tmp
		case 'd':
			// We can parse days as if they were hours then just
			// multiply the result by 24. This will result in some
			// loss of precision, but it should otherwise be fine.
			tmp, err := time.ParseDuration(match[:len(match)-1] + "h")
			if err != nil {
				return ret, err
			}

			ret += tmp * 24
		default:
			return ret, errors.New("Unknown time type")
		}
	}

	return ret, nil
}

func (p *reminderPlugin) RemindCommand(b *seabird.Bot, m *irc.Message) {
	split := strings.SplitN(m.Trailing(), " ", 2)
	if len(split) != 2 {
		b.MentionReply(m, "Not enough args")
		return
	}

	dur, err := p.ParseTime(split[0])
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

	if b.FromChannel(m) {
		// If it was from a channel, we need to prepend the user's name.
		r.Target = m.Params[0]
		r.TargetType = channelTarget
		r.Content = m.Prefix.Name + ": " + r.Content
	}

	err = p.db.Update(func(tx *nut.Tx) error {
		bucket := tx.Bucket("remind_reminders")

		key, innerErr := bucket.NextID()
		if innerErr != nil {
			return innerErr
		}

		r.Key = key

		return bucket.Put(r.Key, r)
	})

	if err != nil {
		b.MentionReply(m, "Failed to store reminder: %s", err)
		return
	}

	b.MentionReply(m, "Event stored")

	logger := b.GetLogger()
	logger.WithField("reminder", r).Debug("Stored reminder")

	p.updateChan <- struct{}{}
}
