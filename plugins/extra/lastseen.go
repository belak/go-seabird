package extra

import (
	"fmt"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/go-xorm/xorm"
	"github.com/lrstanley/girc"

	seabird "github.com/belak/go-seabird"
)

func init() {
	seabird.RegisterPlugin("lastseen", newLastSeenPlugin)
}

type lastSeenPlugin struct {
	db     *xorm.Engine
	logger *logrus.Entry
}

// LastSeen is the xorm model for the lastseen plugin
type LastSeen struct {
	ID      int64
	Channel string
	Nick    string
	Time    time.Time
}

func newLastSeenPlugin(b *seabird.Bot, c *girc.Client, db *xorm.Engine) error {
	p := &lastSeenPlugin{db: db, logger: b.GetLogger()}

	err := p.db.Sync(LastSeen{})
	if err != nil {
		return err
	}

	c.Handlers.AddBg(seabird.PrefixCommand("active"), p.activeCallback)
	c.Handlers.Add(girc.PRIVMSG, p.msgCallback)

	/*
		cm.Event("active", p.activeCallback, &seabird.HelpInfo{
			Usage:       "<nick>",
			Description: "Reports the last time user was seen",
		})

		m.Event("PRIVMSG", p.msgCallback)
	*/

	return nil
}

func (p *lastSeenPlugin) activeCallback(c *girc.Client, e girc.Event) {
	nick := e.Last()
	if nick == "" {
		c.Cmd.ReplyTof(e, "Nick required")
		return
	}

	channel := e.Params[0]

	c.Cmd.ReplyTof(e, "%s", p.getLastSeen(nick, channel))
}

func (p *lastSeenPlugin) getLastSeen(rawNick, rawChannel string) string {
	search := LastSeen{
		Channel: strings.ToLower(rawChannel),
		Nick:    strings.ToLower(rawNick),
	}

	found, err := p.db.Get(&search)
	if err != nil || !found {
		return rawNick + " has not been seen in " + rawChannel
	}

	return rawNick + " was last active on " + formatDate(search.Time) + " at " + formatTime(search.Time)
}

func formatTime(t time.Time) string {
	return fmt.Sprintf("%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
}

func formatDate(t time.Time) string {
	return fmt.Sprintf("%d %s %d", t.Day(), t.Month().String(), t.Year())
}

func (p *lastSeenPlugin) msgCallback(c *girc.Client, e girc.Event) {
	if len(e.Params) < 2 || !e.IsFromChannel() || e.Source.Name == "" {
		return
	}

	nick := e.Source.Name
	channel := e.Params[0]

	p.updateLastSeen(c, nick, channel)
}

// Thanks to @belak for the comments
func (p *lastSeenPlugin) updateLastSeen(c *girc.Client, rawNick, rawChannel string) {
	search := LastSeen{
		Channel: strings.ToLower(rawChannel),
		Nick:    strings.ToLower(rawNick),
	}

	_, err := p.db.Transaction(func(s *xorm.Session) (interface{}, error) {
		found, _ := s.Get(&search)
		if !found {
			search.Time = time.Now()
			return s.Insert(search)
		}

		return s.ID(search.ID).Update(search)
	})

	if err != nil {
		p.logger.WithError(err).Warnf("Failed to update lastseen data for %s in %s", rawNick, rawChannel)
	}
}
