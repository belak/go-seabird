package extra

import (
	"fmt"
	"strings"
	"time"

	"github.com/belak/go-seabird"
	"github.com/belak/nut"
	irc "github.com/go-irc/irc/v2"
)

func init() {
	seabird.RegisterPlugin("lastseen", newLastSeenPlugin)
}

type lastSeenPlugin struct {
	db *nut.DB
}

type lastSeenChannelBucket struct {
	Key   string
	Nicks map[string]time.Time
}

func newLastSeenPlugin(m *seabird.BasicMux, cm *seabird.CommandMux, db *nut.DB) error {
	p := &lastSeenPlugin{db: db}

	err := p.db.EnsureBucket("lastseen")
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
	nick := strings.ToLower(rawNick)

	channelBucket := &lastSeenChannelBucket{
		Key: strings.ToLower(rawChannel),
	}

	err := p.db.View(func(tx *nut.Tx) error {
		bucket := tx.Bucket("lastseen")
		return bucket.Get(channelBucket.Key, channelBucket)
	})
	if err != nil {
		return "Unknown channel"
	}

	var tm time.Time
	var ok bool
	if tm, ok = channelBucket.Nicks[nick]; !ok {
		return "Unknown user"
	}

	return rawNick + " was last active on " + formatDate(tm) + " at " + formatTime(tm)
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
	nick := strings.ToLower(rawNick)

	channelBucket := &lastSeenChannelBucket{
		Key:   strings.ToLower(rawChannel),
		Nicks: make(map[string]time.Time),
	}

	_ = p.db.Update(func(tx *nut.Tx) error {
		bucket := tx.Bucket("lastseen")

		bucket.Get(channelBucket.Key, channelBucket)
		channelBucket.Nicks[nick] = time.Now()
		return bucket.Put(channelBucket.Key, channelBucket)
	})
}
