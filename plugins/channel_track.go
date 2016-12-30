package plugins

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/belak/go-seabird"
	"github.com/belak/irc"
	uuid "github.com/satori/go.uuid"
)

// TODO: Figure out what to do with dead sessions in other
// plugins... maybe a session removal callback.
//
// TODO: Improve logging

func init() {
	seabird.RegisterPlugin("channel_track", newChannelTracker)
}

// Channel is an internal type for representing a channel.
type Channel struct {
	users map[string]bool
}

// HasUser returns true if the user is in the channel, otherwise
// false.
func (c *Channel) HasUser(user string) bool {
	return c.users[user]
}

// User is an type for representing a user.
type User struct {
	channels map[string]bool

	Nick string
	UUID string
}

// Channels returns which channels the user is currently in.
func (u *User) Channels() []string {
	var ret []string
	for k := range u.channels {
		ret = append(ret, k)
	}
	return ret
}

// InChannel returns true if the user is in the channel, otherwise
// false.
func (u *User) InChannel(channel string) bool {
	return u.channels[channel]
}

// ChannelTracker is a simple plugin which is only meant to track what
// channels the bot is in, and what users are in a channel. It also
// provides a uuid mapping to a user, so if a user's nick changes,
// we'll still have a sort of "session" to keep track of them.
type ChannelTracker struct {
	isupport *ISupportPlugin

	// Channels can't be renamed, so it's just a mapping of name to
	// channel object.
	channels map[string]*Channel

	// Users can be renamed so we key them on uuid. There's also a
	// separate nick to uuid mapping.
	users map[string]*User

	// This simply maps the nick to the uuid
	uuids map[string]string
}

func newChannelTracker(bm *seabird.BasicMux, isupport *ISupportPlugin) *ChannelTracker {
	p := &ChannelTracker{
		isupport: isupport,
		channels: make(map[string]*Channel),
		users:    make(map[string]*User),
		uuids:    make(map[string]string),
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

// Public interfaces

// LookupUser will return the User object for the given nick or nil if
// we don't know about this user. The returned value can be stored and
// will track this user even if they change nicks.
func (p *ChannelTracker) LookupUser(user string) *User {
	userUUID, ok := p.uuids[user]
	if !ok {
		return nil
	}

	return p.users[userUUID]
}

// UsersInChannel will return all the users in the given channel name
// or nil if we're not in that channel.
func (p *ChannelTracker) UsersInChannel(channel string) []*User {
	c := p.LookupChannel(channel)
	if c == nil {
		return nil
	}

	var ret []*User
	for userUUID := range c.users {
		ret = append(ret, p.users[userUUID])
	}
	return ret
}

// LookupChannel will return the Channel object for the given channel
// name or nil if we're not in that channel.
func (p *ChannelTracker) LookupChannel(channel string) *Channel {
	return p.channels[channel]
}

// Channels will return all the channel objects this bot knows about.
func (p *ChannelTracker) Channels() []*Channel {
	var ret []*Channel
	for _, v := range p.channels {
		ret = append(ret, v)
	}
	return ret
}

// Private functions

func (p *ChannelTracker) joinCallback(b *seabird.Bot, m *irc.Message) {
	user := m.Prefix.Name
	channel := m.Trailing()

	p.addUserToChannel(b, user, channel)

	fmt.Printf("%s (%s) joined %s\n", user, p.uuids[user], channel)
}

func (p *ChannelTracker) partCallback(b *seabird.Bot, m *irc.Message) {
	user := m.Prefix.Name
	channel := m.Params[0]

	p.removeUserFromChannel(b, user, channel)

	fmt.Printf("%s (%s) left %s\n", user, p.uuids[user], channel)
}

func (p *ChannelTracker) kickCallback(b *seabird.Bot, m *irc.Message) {
	actor := m.Prefix.Name
	user := m.Params[1]
	channel := m.Params[0]

	p.removeUserFromChannel(b, user, channel)

	fmt.Printf("%s (%s) kicked %s (%s) from %s\n", actor, p.uuids[actor], user, p.uuids[user], channel)
}

func (p *ChannelTracker) quitCallback(b *seabird.Bot, m *irc.Message) {
	user := m.Prefix.Name

	p.removeUser(b, user)

	fmt.Printf("%s (%s) quit\n", user, p.uuids[user])
}

func (p *ChannelTracker) nickCallback(b *seabird.Bot, m *irc.Message) {
	oldUser := m.Prefix.Name
	newUser := m.Params[0]

	p.renameUser(b, oldUser, newUser)

	fmt.Printf("%s (%s) changed their name to %s\n", oldUser, p.uuids[newUser], newUser)
}

func (p *ChannelTracker) namesCallback(b *seabird.Bot, m *irc.Message) {
	logger := b.GetLogger()

	// Sample: (qaohv)~&@%+
	prefix, _ := p.isupport.GetRaw("PREFIX")

	// We only care about the symbols
	i := strings.IndexByte(prefix, ')')
	if len(prefix) == 0 || prefix[0] != '(' || i < 0 {
		logger.WithField("prefix", prefix).Warnf("Invalid prefix format")
		return
	}

	prefixes := prefix[i:]

	channel := m.Params[2]
	users := strings.Split(m.Trailing(), " ")
	for _, user := range users {
		user = strings.TrimLeft(user, prefixes)

		// The bot user should be added via JOIN
		if user == b.CurrentNick() {
			continue
		}

		p.addUserToChannel(b, user, channel)

		fmt.Printf("%s (%s) is in channel %s\n", user, p.uuids[user], channel)
	}
}

func (p *ChannelTracker) endOfNamesCallback(b *seabird.Bot, m *irc.Message) {
	channel := m.Params[1]

	fmt.Printf("Got all names for %s\n", channel)
}

// Implementation below

func (p *ChannelTracker) addUserToChannel(b *seabird.Bot, user, channel string) {
	logger := b.GetLogger().WithField("channel", channel)

	// If the current user is joining a channel, we need to add it
	// before adding our user.
	if user == b.CurrentNick() {
		p.addChannel(b, channel)
	}

	// If we're not in this channel, issue a warning and bail.
	c, ok := p.channels[channel]
	if !ok {
		logger.Warn("Bot not in channel")
		return
	}

	// If there's no mapping, add it
	userUUID, ok := p.uuids[user]
	if !ok {
		userUUID = uuid.NewV4().String()
		p.uuids[user] = userUUID
	}

	// Add the user if they don't exist.
	u, ok := p.users[userUUID]
	if !ok {
		u = &User{
			Nick:     user,
			UUID:     userUUID,
			channels: make(map[string]bool),
		}
		p.users[userUUID] = u
	}

	if _, ok := u.channels[channel]; ok {
		logger.Warn("User is already in channel")
	}

	u.channels[channel] = true
	c.users[userUUID] = true
}

func (p *ChannelTracker) removeUserFromChannel(b *seabird.Bot, user, channel string) {
	logger := b.GetLogger().WithField("channel", channel)

	if user == b.CurrentNick() {
		p.removeChannel(b, channel)
	} else {
		userUUID, ok := p.uuids[user]
		if !ok {
			logger.Warn("Can't remove unknown user")
			return
		}

		u := p.users[userUUID]
		if _, ok := u.channels[channel]; !ok {
			logger.Warn("Not in channel")
		} else {
			delete(u.channels, channel)
		}

		if len(u.channels) == 0 {
			p.removeUser(b, user)
		}
	}
}

func (p *ChannelTracker) addChannel(b *seabird.Bot, channel string) {
	logger := b.GetLogger()

	// TODO: This can get called if we're already in the channel. This
	// will cause some invalid warning calls.

	// Woo! We joined a channel!
	if _, ok := p.channels[channel]; ok {
		logger.WithField("channel", channel).Warn("Already in channel")
	}

	p.channels[channel] = &Channel{
		users: make(map[string]bool),
	}
}

func (p *ChannelTracker) removeChannel(b *seabird.Bot, channel string) {
	logger := b.GetLogger()

	c, ok := p.channels[channel]
	if !ok {
		logger.Warn("Not in channel")
		return
	}

	// Remove all users currently in this channel from this channel.
	for userUUID := range c.users {
		u := p.users[userUUID]
		delete(u.channels, channel)

		// If this user has no more channels, they need to be removed.
		if len(u.channels) == 0 {
			p.removeUser(b, u.Nick)
		}
	}

	// Clean up the channel object
	c.users = make(map[string]bool)

	// Remove the channel from tracking
	delete(p.channels, channel)
}

func (p *ChannelTracker) removeUser(b *seabird.Bot, user string) {
	logger := b.GetLogger()

	userUUID, ok := p.uuids[user]
	if !ok {
		logger.Warn("User does not exist")
		return
	}

	// We need to clear out the channels and Nick to show this session
	// is invalid.
	u := p.users[userUUID]
	u.Nick = ""
	for channel := range u.channels {
		delete(p.channels[channel].users, userUUID)
	}
	u.channels = make(map[string]bool)

	// Now that the User is empty, delete all internal traces.
	delete(p.uuids, user)
	delete(p.users, userUUID)
}

func (p *ChannelTracker) renameUser(b *seabird.Bot, oldNick, newNick string) {
	userUUID, ok := p.uuids[oldNick]
	if !ok {
		logger := b.GetLogger()
		logger.WithFields(logrus.Fields{
			"oldNick": oldNick,
			"newNick": newNick,
		}).Warn("Can't rename user that doesn't exist")
		return
	}

	// Rename the user object
	u := p.users[userUUID]
	u.Nick = newNick

	// Swap where the UUID points to
	delete(p.uuids, oldNick)
	p.uuids[newNick] = userUUID
}
