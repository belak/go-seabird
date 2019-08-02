package extra

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/go-xorm/xorm"
	"github.com/lrstanley/girc"

	seabird "github.com/belak/go-seabird"
)

func init() {
	seabird.RegisterPlugin("remind", newReminderPlugin)
}

var timeRegexp = regexp.MustCompile(`\d+[smhd]`)

type reminderPlugin struct {
	db     *xorm.Engine
	logger *logrus.Entry

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

func newReminderPlugin(b *seabird.Bot, c *girc.Client, db *xorm.Engine) error {
	p := &reminderPlugin{
		db:         db,
		logger:     b.GetLogger(),
		updateChan: make(chan struct{}, 1),
	}

	err := p.db.Sync(Reminder{})
	if err != nil {
		return err
	}

	c.Handlers.AddBg(girc.RPL_WELCOME, p.InitialDispatch)
	c.Handlers.AddBg(seabird.PrefixCommand("remind"), p.RemindCommand)

	/*
		cm.Event("remind", p.RemindCommand, &seabird.HelpInfo{
			Usage:       "<duration> <message>",
			Description: "Remind yourself to do something.",
		})
	*/

	return nil
}

func (p *reminderPlugin) nextReminder() (*Reminder, error) {
	// Find the next reminder we'll have to send
	//
	// TODO: This should only grab reminders for users who are online and
	// channels we are in
	r := &Reminder{}
	_, err := p.db.OrderBy("reminder_time ASC").Get(r)
	if r.ID == 0 {
		r = nil
	}
	return r, err
}

func (p *reminderPlugin) remindLoop(c *girc.Client) {
	p.logger.Info("Starting reminder loop")

	for {
		r, err := p.nextReminder()
		if err != nil {
			p.logger.WithError(err).Error("Transaction failure. Exiting loop.")
			return
		}

		var timer <-chan time.Time
		if r != nil {
			p.logger.WithField("reminder", r).Debug("Next reminder")

			waitDur := r.ReminderTime.Sub(time.Now())
			if waitDur <= 0 {
				p.dispatch(c, r)
				continue
			}

			timer = time.After(waitDur)
		}

		select {
		case <-timer:
			p.dispatch(c, r)
		case <-p.updateChan:
			continue
		}
	}
}

func (p *reminderPlugin) dispatch(c *girc.Client, r *Reminder) {
	logger := p.logger.WithField("reminder", r)

	// Send the message
	c.Cmd.Message(r.Target, r.Content)

	// Nuke the reminder now that it's been sent
	_, err := p.db.Delete(r)
	if err != nil {
		logger.WithError(err).Error("Failed to remove reminder")
	}

	logger.Debug("Dispatched reminder")
}

// InitialDispatch is used to send private messages to users on connection. We
// can't queue up the channels yet because we haven't joined them.
func (p *reminderPlugin) InitialDispatch(c *girc.Client, e girc.Event) {
	go p.remindLoop(c)
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

func (p *reminderPlugin) RemindCommand(c *girc.Client, e girc.Event) {
	split := strings.SplitN(e.Last(), " ", 2)
	if len(split) != 2 {
		c.Cmd.ReplyTof(e, "Not enough args")
		return
	}

	dur, err := p.ParseTime(split[0])
	if err != nil {
		c.Cmd.ReplyTof(e, "Invalid duration: %s", err)
		return
	}

	r := &Reminder{
		Target:       e.Source.Name,
		TargetType:   privateTarget,
		Content:      split[1],
		ReminderTime: time.Now().Add(dur),
	}

	if e.IsFromChannel() {
		// If it was from a channel, we need to prepend the user's name.
		r.Target = e.Params[0]
		r.TargetType = channelTarget
		r.Content = e.Source.Name + ": " + r.Content
	}

	_, err = p.db.Insert(r)
	if err != nil {
		c.Cmd.ReplyTof(e, "Failed to store reminder: %s", err)
		return
	}

	c.Cmd.ReplyTof(e, "Event stored")

	p.logger.WithField("reminder", r).Debug("Stored reminder")

	p.updateChan <- struct{}{}
}
