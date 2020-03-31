package lastseen

import (
	"fmt"
	"strings"
	"time"

	"xorm.io/xorm"

	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/plugins/extra/db"
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

func newLastSeenPlugin(b *seabird.Bot) error {
	if err := b.EnsurePlugin("db"); err != nil {
		return err
	}

	p := &lastSeenPlugin{
		db: db.CtxDB(b.Context()),
	}

	if err := p.db.Sync(LastSeen{}); err != nil {
		return err
	}

	bm := b.BasicMux()
	cm := b.CommandMux()

	cm.Event("active", p.activeCallback, &seabird.HelpInfo{
		Usage:       "<nick>",
		Description: "Reports the last time user was seen",
	})

	bm.Event("PRIVMSG", p.msgCallback)

	return nil
}

func (p *lastSeenPlugin) activeCallback(r *seabird.Request) {
	nick := r.Message.Trailing()
	if nick == "" {
		r.MentionReplyf("Nick required")
		return
	}

	channel := r.Message.Params[0]

	r.MentionReplyf("%s", p.getLastSeen(nick, channel))
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

func (p *lastSeenPlugin) msgCallback(r *seabird.Request) {
	if len(r.Message.Params) < 2 || !r.FromChannel() || r.Message.Prefix.Name == "" {
		return
	}

	nick := r.Message.Prefix.Name
	channel := r.Message.Params[0]

	p.updateLastSeen(r, nick, channel)
}

// Thanks to @belak for the comments
func (p *lastSeenPlugin) updateLastSeen(r *seabird.Request, rawNick, rawChannel string) {
	l := r.GetLogger("lastseen")

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
		l.WithError(err).Warnf("Failed to update lastseen data for %s in %s", rawNick, rawChannel)
	}
}
