package extra

import (
	"strings"
	"sync"
	"time"
	"bytes"
	"unicode"
	"strconv"
	"fmt"
	"errors"

	"github.com/belak/go-seabird"
	"github.com/belak/irc"
	"github.com/belak/nut"
)

func init() {
	seabird.RegisterPlugin("remind", newreminderPlugin)
}

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

type timeData struct {
	Secs  float64
	Mins  float64
	Hours float64
	Days  float64
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

// Parses the text string and turns it into a time.Duration
func (p *reminderPlugin) ParseTime(timeStr string) (time.Duration, error) {
	buf := bytes.NewBufferString(timeStr)
	parsed := &timeData{}

	var tmp string
	var val float64
	var isFloat bool
	for buf.Len() > 0 {
		// Read next character
		curChar, size, err := buf.ReadRune()
		if err != nil || size > 1 {
			return -1, err
		}

		// Fix uppercase
		if curChar >= 'A' && curChar <= 'Z' {
			curChar = unicode.ToLower(curChar)
		}

		// Parse previous number if we're at a letter
		if curChar >= 'a' && curChar <= 'z' {
			if tmp == "" || tmp == "-" || tmp == "." {
				return -1, errors.New("Blank value passed.")
			}

			val, err = strconv.ParseFloat(tmp, 64)
			if err != nil {
				return -1, err
			}

			tmp = ""
		}

		// Store parsed values
		switch (curChar) {
		case 'd':
			parsed.Days = val
		case 'h':
			parsed.Hours = val
		case 'm':
			parsed.Mins = val
		case 's':
			parsed.Secs = val
		case '.':
			if !isFloat {
				isFloat = true
				tmp += "."
			} else {
				return -1, errors.New("Too many decimal points specified.")
			}
		default:
			if curChar >= '0' && curChar <= '9' {
				tmp += fmt.Sprintf("%c", curChar)
			} else if tmp == "" && curChar == '-' {
				// Only allow - at the beginning of numbers
				tmp += "-"
			} else {
				return -1, errors.New("Invalid character specified.")
			}
		}
	}

	// Calculate seconds
	var nsecs time.Duration
	nsecs = time.Duration(((parsed.Days * 86400) + (parsed.Hours * 3600) + (parsed.Mins * 60) + parsed.Secs) * 1000000000)
	if nsecs < 0 {
		return -1, errors.New("Setting reminders in the past is pointless.")
	}

//	fmt.Printf("parsed = %+v\nseconds = %d\n", parsed, nsecs)

	return nsecs, nil
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

	if m.FromChannel() {
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
