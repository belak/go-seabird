package extra

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-xorm/xorm"

	seabird "github.com/belak/go-seabird"
	irc "github.com/go-irc/irc/v2"
)

func init() {
	seabird.RegisterPlugin("lastseen", newLastSeenPlugin)
}

type lastSeenPlugin struct {
	db *xorm.Engine
}

// LastSeen is the xorm model for the lastseen plugin
type LastSeen struct {
	ID      int64
	Channel string
	Nick    string
	Time    time.Time
}

func newLastSeenPlugin(m *seabird.BasicMux, cm *seabird.CommandMux, db *xorm.Engine) error {
	p := &lastSeenPlugin{db: db}

	err := p.db.Sync(LastSeen{})
	if err != nil {
		return err
	}

	cm.Event("active", p.activeCallback, &seabird.HelpInfo{
		Usage:       "<nick>",
		Description: "Reports the last time user was seen",
	})

	m.Event("PRIVMSG", p.msgCallback)

	return nil
}

func (p *lastSeenPlugin) activeCallback(b *seabird.Bot, m *irc.Message) {
	nick := m.Trailing()
	if nick == "" {
		b.MentionReply(m, "Nick required")
		return
	}

	channel := m.Params[0]

	b.MentionReply(m, "%s", p.getLastSeen(nick, channel))
}

func (p *lastSeenPlugin) getLastSeen(rawNick, rawChannel string) string {
	search := LastSeen{
		Channel: strings.ToLower(rawChannel),
		Nick:    strings.ToLower(rawNick),
	}

	_, err := p.db.Get(&search)
	if err != nil {
		return rawNick + " has not been seen in" + rawChannel
	}

	return rawNick + " was last active on " + formatDate(search.Time) + " at " + formatTime(search.Time)
}

func formatTime(t time.Time) string {
	return fmt.Sprintf("%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
}

func formatDate(t time.Time) string {
	return fmt.Sprintf("%d %s %d", t.Day(), t.Month().String(), t.Year())
}

func (p *lastSeenPlugin) msgCallback(b *seabird.Bot, m *irc.Message) {
	if len(m.Params) < 2 || !b.FromChannel(m) || m.Prefix.Name == "" {
		return
	}

	nick := m.Prefix.Name
	channel := m.Params[0]

	p.updateLastSeen(nick, channel)
}

// Thanks to @belak for the comments
func (p *lastSeenPlugin) updateLastSeen(rawNick, rawChannel string) {
	search := LastSeen{
		Channel: strings.ToLower(rawChannel),
		Nick:    strings.ToLower(rawNick),
	}

	_, _ = p.db.Transaction(func(s *xorm.Session) (interface{}, error) {
		found, _ := s.Get(&search)
		if !found {
			search.Time = time.Now()
			return s.Insert(search)
		}

		return s.ID(search.ID).Update(search)
	})
}
