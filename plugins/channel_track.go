package plugins

import (
	"fmt"
	"strings"

	"github.com/belak/go-seabird"
	"github.com/belak/irc"
)

func init() {
	seabird.RegisterPlugin("channel_track", newChannelTracker)
}

// ChannelTracker is a simple plugin which is only meant to track what
// channels the bot is in, and what users are in a channel. It also
// provides a uuid mapping to a user, so if a user's nick changes,
// we'll still have a sort of "session" to keep track of them.
type ChannelTracker struct {
	isupport *ISupportPlugin
}

// TODO:
// UserInChannel
// GetUsersInChannel
// BotInChannel
// GetBotChannels

func newChannelTracker(bm *seabird.BasicMux, isupport *ISupportPlugin) *ChannelTracker {
	p := &ChannelTracker{
		isupport: isupport,
	}

	bm.Event("JOIN", p.joinCallback)
	bm.Event("PART", p.partCallback)
	bm.Event("KICK", p.kickCallback)
	bm.Event("QUIT", p.quitCallback)
	bm.Event("NICK", p.nickCallback)

	bm.Event("353", p.namesCallback)
	bm.Event("366", p.endOfNamesCallback)

	return p
}

func (p *ChannelTracker) joinCallback(b *seabird.Bot, m *irc.Message) {
	user := m.Prefix.Name
	channel := m.Trailing()

	fmt.Printf("%s joined %s\n", user, channel)
}

func (p *ChannelTracker) partCallback(b *seabird.Bot, m *irc.Message) {
	user := m.Prefix.Name
	channel := m.Params[0]

	fmt.Printf("%s left %s\n", user, channel)
}

func (p *ChannelTracker) kickCallback(b *seabird.Bot, m *irc.Message) {
	actor := m.Prefix.Name
	user := m.Params[1]
	channel := m.Params[0]

	fmt.Printf("%s kicked %s from %s\n", actor, user, channel)
}

func (p *ChannelTracker) quitCallback(b *seabird.Bot, m *irc.Message) {
	user := m.Prefix.Name

	fmt.Printf("%s quit\n", user)
}

func (p *ChannelTracker) nickCallback(b *seabird.Bot, m *irc.Message) {
	oldUser := m.Prefix.Name
	newUser := m.Params[0]

	fmt.Printf("%s changed their name to %s\n", oldUser, newUser)
}

func (p *ChannelTracker) namesCallback(b *seabird.Bot, m *irc.Message) {
	logger := b.GetLogger()

	// Sample: (qaohv)~&@%+
	prefix, ok := p.isupport.GetRaw("PREFIX")
	if !ok {
		// TODO: Put default value in isupport plugin
		prefix = "(ov)@+"
	}

	// We only care about the symbols
	i := strings.IndexByte(prefix, ')')
	if i < 0 {
		logger.WithField("prefix", prefix).Warnf("Invalid prefix format")
		return
	}

	prefixes := prefix[i:]

	channel := m.Params[2]
	users := strings.Split(m.Trailing(), " ")
	for _, user := range users {
		user = strings.TrimLeft(user, prefixes)
		fmt.Printf("%s is in channel %s\n", user, channel)
	}

	fmt.Println(m)
}

func (p *ChannelTracker) endOfNamesCallback(b *seabird.Bot, m *irc.Message) {
	channel := m.Params[1]

	fmt.Printf("Got all names for %s\n", channel)
}
