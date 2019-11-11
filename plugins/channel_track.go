package plugins

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/internal"
)

func init() {
	seabird.RegisterPlugin("channel_track", newChannelTracker)
}

const contextKeyChannelTracker = internal.ContextKey("seabird-channel-tracker")

func CtxChannelTracker(ctx context.Context) *ChannelTracker {
	return ctx.Value(contextKeyChannelTracker).(*ChannelTracker)
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
	channels map[string]map[rune]bool
	Nick     string
	UUID     string
}

// Channels returns which channels the user is currently in.
func (u *User) Channels() []string {
	var ret []string
	for k := range u.channels {
		ret = append(ret, k)
	}

	return ret
}

// ModesInChannel returns a mapping of channel modes to a bool indicating if
// it's on or not for this user in this channel.
func (u *User) ModesInChannel(channel string) map[rune]bool {
	ret, ok := u.channels[channel]
	if !ok {
		ret = make(map[rune]bool)
	}

	return ret
}

// InChannel returns true if the user is in the channel, otherwise
// false.
func (u *User) InChannel(channel string) bool {
	_, ok := u.channels[channel]
	return ok
}

// ChannelTracker is a simple plugin which is only meant to track what
// channels the bot is in, and what users are in a channel. It also
// provides a uuid mapping to a user, so if a user's nick changes,
// we'll still have a sort of "session" to keep track of them.
type ChannelTracker struct {
	// Notes for internal fields. Be very careful when modifying the
	// state. Because we control all of this, it is valid to make the
	// assumption that if a user is in p.uuids, it will be possible to
	// find them in p.users.

	// Channels can't be renamed, so it's just a mapping of name to
	// channel object.
	channels map[string]*Channel

	// Users can be renamed so we key them on uuid. There's also a
	// separate nick to uuid mapping.
	users map[string]*User

	// This simply maps the nick to the uuid
	uuids map[string]string

	// Session cleanup callbacks
	cleanupCallbacks []func(u *User)
}

func newChannelTracker(b *seabird.Bot) error {
	bm := b.BasicMux()

	// TODO: ensure isupport loaded

	p := &ChannelTracker{
		channels: make(map[string]*Channel),
		users:    make(map[string]*User),
		uuids:    make(map[string]string),
	}

	bm.Event("JOIN", p.joinCallback)
	bm.Event("PART", p.partCallback)
	bm.Event("KICK", p.kickCallback)
	bm.Event("QUIT", p.quitCallback)
	bm.Event("NICK", p.nickCallback)
	bm.Event("MODE", p.modeCallback)

	bm.Event("352", p.whoCallback)
	bm.Event("353", p.namesCallback)
	bm.Event("366", p.endOfNamesCallback)

	b.SetValue(contextKeyChannelTracker, p)

	return nil
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

// RegisterSessionCleanupCallback lets you register a function to be
// called when a session is removed.
func (p *ChannelTracker) RegisterSessionCleanupCallback(f func(u *User)) {
	p.cleanupCallbacks = append(p.cleanupCallbacks, f)
}

// Private functions

func (p *ChannelTracker) joinCallback(ctx context.Context, r *seabird.Request) {
	user := r.Message.Prefix.Name
	channel := r.Message.Trailing()

	p.addUserToChannel(ctx, user, channel)

	//fmt.Printf("%s (%s) joined %s\n", user, p.uuids[user], channel)
} //nolint:wsl

func (p *ChannelTracker) partCallback(ctx context.Context, r *seabird.Request) {
	user := r.Message.Prefix.Name
	channel := r.Message.Params[0]

	p.removeUserFromChannel(ctx, user, channel)

	//fmt.Printf("%s (%s) left %s\n", user, p.uuids[user], channel)
} //nolint:wsl

func (p *ChannelTracker) kickCallback(ctx context.Context, r *seabird.Request) {
	//actor := m.Prefix.Name
	user := r.Message.Params[1]
	channel := r.Message.Params[0]

	p.removeUserFromChannel(ctx, user, channel)

	//fmt.Printf("%s (%s) kicked %s (%s) from %s\n", actor, p.uuids[actor], user, p.uuids[user], channel)
} //nolint:wsl

func (p *ChannelTracker) quitCallback(ctx context.Context, r *seabird.Request) {
	user := r.Message.Prefix.Name

	p.removeUser(ctx, user)

	//fmt.Printf("%s (%s) quit\n", user, p.uuids[user])
} //nolint:wsl

func (p *ChannelTracker) nickCallback(ctx context.Context, r *seabird.Request) {
	oldUser := r.Message.Prefix.Name
	newUser := r.Message.Params[0]

	p.renameUser(ctx, oldUser, newUser)

	//fmt.Printf("%s (%s) changed their name to %s\n", oldUser, p.uuids[newUser], newUser)
} //nolint:wsl

func (p *ChannelTracker) modeCallback(ctx context.Context, r *seabird.Request) {
	// We only care about MODE messages where a specific user is
	// changed.
	if len(r.Message.Params) < 3 {
		return
	}

	logger := seabird.CtxLogger(ctx)

	channel := r.Message.Params[0]
	target := r.Message.Params[2]

	// Ensure we know about this user and this channel
	u := p.LookupUser(target)
	c := p.LookupChannel(channel)

	if u == nil || c == nil {
		logger.Warnf("Got MODE callback for %s on %s but we aren't tracking both", target, channel)
		return
	}

	// Just send a WHO request and clear out the modes for this user because
	// mode parsing is hard.
	u.channels[channel] = make(map[rune]bool)

	r.Writef("WHO :%s", target)
}

func (p *ChannelTracker) whoCallback(ctx context.Context, r *seabird.Request) {
	// Filter out broken messages
	if len(r.Message.Params) < 7 {
		return
	}

	prefixes, ok := p.getSymbolToPrefixMapping(ctx)
	if !ok {
		return
	}

	var (
		channel = r.Message.Params[0]
		nick    = r.Message.Params[4]
		modes   = r.Message.Params[5]
	)

	logger := seabird.CtxLogger(ctx)

	u := p.LookupUser(nick)
	c := p.LookupChannel(channel)

	if u == nil || c == nil {
		logger.Warnf("Got WHO callback for %s on %s but we aren't tracking both", nick, channel)
		return
	}

	// Modes starts with H/G for here/gone, so we skip that because we don't
	// care too much about tracking it for now.
	userPrefixes := modes[1:]

	// Clear out the modes and reset them
	u.channels[channel] = make(map[rune]bool)

	for _, v := range userPrefixes {
		mode := prefixes[v]
		u.channels[channel][mode] = true
	}
}

// getSymbolToPrefixMapping gets the isupport info from the bot and
// parses prefix into a mapping of the symbol to the mode. Eventually
// this should be moved into the isupport plugin with a few more prefix
// helper functions.
func (p *ChannelTracker) getSymbolToPrefixMapping(ctx context.Context) (map[rune]rune, bool) {
	logger := seabird.CtxLogger(ctx)

	isupport := CtxISupport(ctx)

	// Sample: (qaohv)~&@%+
	prefix, _ := isupport.GetRaw("PREFIX")

	logger = logger.WithField("prefix", prefix)

	// We only care about the symbols
	i := strings.IndexByte(prefix, ')')
	if len(prefix) == 0 || prefix[0] != '(' || i < 0 {
		logger.Warnf("Invalid prefix format")
		return nil, false
	}

	// We loop through the string using range so we get bytes, then we throw the
	// two results together in the map.
	var symbols []rune // ~&@%+
	for _, r := range prefix[i+1:] {
		symbols = append(symbols, r)
	}

	var modes []rune // qaohv
	for _, r := range prefix[1:i] {
		modes = append(modes, r)
	}

	if len(modes) != len(symbols) {
		logger.WithFields(logrus.Fields{
			"modes":   modes,
			"symbols": symbols,
		}).Warnf("Mismatched modes and symbols")

		return nil, false
	}

	prefixes := make(map[rune]rune)
	for k := range symbols {
		prefixes[symbols[k]] = modes[k]
	}

	return prefixes, true
}

func (p *ChannelTracker) namesCallback(ctx context.Context, r *seabird.Request) {
	prefixes, ok := p.getSymbolToPrefixMapping(ctx)
	if !ok {
		return
	}

	logger := seabird.CtxLogger(ctx)

	channel := r.Message.Params[2]

	users := strings.Split(strings.TrimSpace(r.Message.Trailing()), " ")
	for _, user := range users {
		i := strings.IndexFunc(user, func(r rune) bool {
			_, ok := prefixes[r]
			return !ok
		})

		var userPrefixes string
		if i != -1 {
			userPrefixes = user[:i]
			user = user[i:]
		}

		// The bot user should be added via JOIN
		if user == seabird.CtxCurrentNick(ctx) {
			continue
		}

		p.addUserToChannel(ctx, user, channel)

		u := p.LookupUser(user)
		if u == nil {
			continue
		}

		// Clear out the modes and reset them
		u.channels[channel] = make(map[rune]bool)

		for _, v := range userPrefixes {
			mode := prefixes[v]
			u.channels[channel][mode] = true
		}

		logger.WithFields(logrus.Fields{
			"user":    user,
			"channel": channel,
			"modes":   u.channels[channel],
		}).Debug("User modes updated")
	}
}

func (p *ChannelTracker) endOfNamesCallback(ctx context.Context, r *seabird.Request) {
	channel := r.Message.Params[1]

	fmt.Printf("Got all names for %s\n", channel)
}

// Implementation below

func (p *ChannelTracker) addUserToChannel(ctx context.Context, user, channel string) {
	logger := seabird.CtxLogger(ctx).WithFields(logrus.Fields{
		"channel": channel,
		"user":    user,
	})

	// If the current user is joining a channel, we need to add it
	// before adding our user.
	if user == seabird.CtxCurrentNick(ctx) {
		p.addChannel(ctx, channel)
	}

	// If we're not in this channel, issue a warning and bail.
	c, ok := p.channels[channel]
	if !ok {
		logger.Warn("Error adding user: bot not in channel")
		return
	}

	u := p.LookupUser(user)
	if u == nil {
		u = &User{
			Nick:     user,
			UUID:     uuid.Must(uuid.NewRandom()).String(),
			channels: make(map[string]map[rune]bool),
		}
		p.users[u.UUID] = u
		p.uuids[user] = u.UUID
	}

	logger = logger.WithFields(logrus.Fields{
		"user": user,
		"uuid": u.UUID,
	})

	if _, ok := u.channels[channel]; ok {
		logger.Warn("User already in channel")
		return
	}

	u.channels[channel] = make(map[rune]bool)
	c.users[u.UUID] = true

	logger.Info("User added to channel")
}

func (p *ChannelTracker) removeUserFromChannel(ctx context.Context, user, channel string) {
	logger := seabird.CtxLogger(ctx).WithField("channel", channel)

	if user == seabird.CtxCurrentNick(ctx) {
		p.removeChannel(ctx, channel)
	} else {
		u := p.LookupUser(user)
		if u == nil {
			logger.Warn("Can't remove unknown user")
			return
		}

		logger = logger.WithField("userUUID", u.UUID)

		if _, ok := u.channels[channel]; !ok {
			logger.Warn("Can only remove users from users they are in")
			return
		}

		delete(u.channels, channel)

		logger.Info("Removing user from channel")

		if len(u.channels) == 0 {
			p.removeUser(ctx, user)
		}

		logger.Info("Removed user from channel")
	}
}

func (p *ChannelTracker) addChannel(ctx context.Context, channel string) {
	logger := seabird.CtxLogger(ctx).WithField("channel", channel)

	if _, ok := p.channels[channel]; ok {
		logger.Warn("Already in channel")
		return
	}

	p.channels[channel] = &Channel{
		users: make(map[string]bool),
	}

	logger.Info("Added channel")
}

func (p *ChannelTracker) removeChannel(ctx context.Context, channel string) {
	logger := seabird.CtxLogger(ctx).WithField("channel", channel)

	c, ok := p.channels[channel]
	if !ok {
		logger.Warn("Can only remove channels we are in")
		return
	}

	// We only do this for channels because it can have side effects
	logger.Info("Removing channel")

	// Remove all users currently in this channel from this channel.
	for userUUID := range c.users {
		u := p.users[userUUID]
		delete(u.channels, channel)

		// If this user has no more channels, they need to be removed.
		if len(u.channels) == 0 {
			p.removeUser(ctx, u.Nick)
		}
	}

	// Clean up the channel object
	c.users = make(map[string]bool)

	// Remove the channel from tracking
	delete(p.channels, channel)

	logger.Info("Removed channel")
}

func (p *ChannelTracker) removeUser(ctx context.Context, user string) {
	logger := seabird.CtxLogger(ctx).WithField("user", user)

	u := p.LookupUser(user)
	if u == nil {
		logger.Warn("User does not exist")
		return
	}

	// We need to clear out the channels and Nick to show this session
	// is invalid.
	u.Nick = ""

	for channel := range u.channels {
		delete(p.channels[channel].users, u.UUID)
	}

	u.channels = make(map[string]map[rune]bool)

	// Now that the User is empty, delete all internal traces.
	delete(p.uuids, user)
	delete(p.users, u.UUID)

	// Run any cleanup callbacks
	for _, f := range p.cleanupCallbacks {
		f(u)
	}

	logger.Info("Removed user")
}

func (p *ChannelTracker) renameUser(ctx context.Context, oldNick, newNick string) {
	logger := seabird.CtxLogger(ctx).WithFields(logrus.Fields{
		"oldNick": oldNick,
		"newNick": newNick,
	})

	u := p.LookupUser(oldNick)
	if u == nil {
		logger.Warn("Can't rename user that doesn't exist")
		return
	}

	logger = logger.WithField("userUUID", u.UUID)

	// Rename the user object
	u.Nick = newNick

	// Swap where the UUID points to
	delete(p.uuids, oldNick)
	p.uuids[newNick] = u.UUID

	logger.Info("Renamed user")
}
