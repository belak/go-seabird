package extra

import (
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	seabird "github.com/belak/go-seabird"
	"xorm.io/xorm"
	irc "gopkg.in/irc.v3"
)

func init() {
	seabird.RegisterPlugin("remind", newReminderPlugin)
}

var timeRegexp = regexp.MustCompile(`\d+[smhd]`)

type reminderPlugin struct {
	db *xorm.Engine

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

// Reminder represents the xorm model for the reminder plugin
type Reminder struct {
	ID           int64
	Target       string
	TargetType   targetType
	Content      string
	ReminderTime time.Time
}

func newReminderPlugin(b *seabird.Bot) error {
	bm := b.BasicMux()
	cm := b.CommandMux()

	if err := b.EnsurePlugin("db"); err != nil {
		return err
	}

	p := &reminderPlugin{
		roomLock:   &sync.Mutex{},
		rooms:      make(map[string]bool),
		updateChan: make(chan struct{}, 1),

		db: CtxDB(b.Context()),
	}

	err := p.db.Sync(Reminder{})
	if err != nil {
		return err
	}

	bm.Event("001", p.InitialDispatch)
	bm.Event("JOIN", p.joinHandler)
	bm.Event("PART", p.partHandler)
	bm.Event("KICK", p.kickHandler)

	cm.Event("remind", p.RemindCommand, &seabird.HelpInfo{
		Usage:       "<duration> <message>",
		Description: "Remind yourself to do something.",
	})

	return nil
}

func (p *reminderPlugin) joinHandler(r *seabird.Request) {
	if r.Message.Prefix.Name != r.CurrentNick() {
		return
	}

	p.roomLock.Lock()
	defer p.roomLock.Unlock()
	p.rooms[r.Message.Params[0]] = true

	p.updateChan <- struct{}{}
}

func (p *reminderPlugin) partHandler(r *seabird.Request) {
	if r.Message.Prefix.Name != r.CurrentNick() {
		return
	}

	p.roomLock.Lock()
	defer p.roomLock.Unlock()
	delete(p.rooms, r.Message.Params[0])

	p.updateChan <- struct{}{}
}

func (p *reminderPlugin) kickHandler(r *seabird.Request) {
	if r.Message.Params[1] != r.CurrentNick() {
		return
	}

	p.roomLock.Lock()
	defer p.roomLock.Unlock()
	delete(p.rooms, r.Message.Params[0])

	p.updateChan <- struct{}{}
}

func (p *reminderPlugin) nextReminder() (*Reminder, error) {
	// Find the next reminder we'll have to send
	r := &Reminder{}
	_, err := p.db.OrderBy("reminder_time ASC").Get(r)

	if r.ID == 0 {
		r = nil
	}

	return r, err
}

func (p *reminderPlugin) remindLoop(r *seabird.Request) {
	logger := r.GetLogger("remind")

	logger.Info("Starting reminder loop")

	// TODO: this should use the bot, not the request, as they're scoped
	// differently.

	for {
		reminder, err := p.nextReminder()
		if err != nil {
			logger.WithError(err).Error("Transaction failure. Exiting loop.")
			return
		}

		var timer <-chan time.Time

		if reminder != nil {
			logger.WithField("reminder", reminder).Debug("Next reminder")

			waitDur := time.Until(reminder.ReminderTime)
			if waitDur <= 0 {
				p.dispatch(r, reminder)
				continue
			}

			timer = time.After(waitDur)
		}

		select {
		case <-timer:
			p.dispatch(r, reminder)
		case <-p.updateChan:
			continue
		}
	}
}

func (p *reminderPlugin) dispatch(r *seabird.Request, reminder *Reminder) {
	logger := r.GetLogger("remind").WithField("reminder", r)

	// Send the message
	r.WriteMessage(&irc.Message{
		Prefix:  &irc.Prefix{},
		Command: "PRIVMSG",
		Params:  []string{reminder.Target, reminder.Content},
	})

	// Nuke the reminder now that it's been sent
	_, err := p.db.Delete(reminder)
	if err != nil {
		logger.WithError(err).Error("Failed to remove reminder")
	}

	logger.Debug("Dispatched reminder")
}

// InitialDispatch is used to send private messages to users on connection. We
// can't queue up the channels yet because we haven't joined them.
func (p *reminderPlugin) InitialDispatch(r *seabird.Request) {
	go p.remindLoop(r)
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

func (p *reminderPlugin) RemindCommand(r *seabird.Request) {
	split := strings.SplitN(r.Message.Trailing(), " ", 2)
	if len(split) != 2 {
		r.MentionReply("Not enough args")
		return
	}

	dur, err := p.ParseTime(split[0])
	if err != nil {
		r.MentionReply("Invalid duration: %s", err)
		return
	}

	rem := &Reminder{
		Target:       r.Message.Prefix.Name,
		TargetType:   privateTarget,
		Content:      split[1],
		ReminderTime: time.Now().Add(dur),
	}

	if r.FromChannel() {
		// If it was from a channel, we need to prepend the user's name.
		rem.Target = r.Message.Params[0]
		rem.TargetType = channelTarget
		rem.Content = r.Message.Prefix.Name + ": " + rem.Content
	}

	_, err = p.db.Insert(rem)
	if err != nil {
		r.MentionReply("Failed to store reminder: %s", err)
		return
	}

	r.MentionReply("Event stored")

	logger := r.GetLogger("remind")
	logger.WithField("reminder", rem).Debug("Stored reminder")

	p.updateChan <- struct{}{}
}
