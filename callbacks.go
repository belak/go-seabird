package seabird

import (
	"github.com/thoj/go-ircevent"

	"fmt"
	"strings"
	"unicode"
)

// Connect handler
func (b *Bot) connect(e *irc.Event) {
	b.UserLock.Lock()
	defer b.UserLock.Unlock()

	// Reset our user map
	b.Users = make(map[string]*User)

	// Run all our raw
	for _, v := range b.Config.Cmds {
		b.Conn.SendRaw(v)
	}
}

// Msg handler
func (b *Bot) msg(e *irc.Event) {
	nick := b.Conn.GetNick()

	// TODO: Handle actual mentions?

	// This callback only handles stuff we've added callbacks for
	if strings.HasPrefix(e.Message, b.Config.Prefix) {
		msgParts := strings.SplitN(e.Message, " ", 2)
		switch len(msgParts) {
		case 2:
			e.Message = msgParts[1]
		case 1:
			e.Message = ""
		}
		cmd := msgParts[0][len(b.Config.Prefix):]
		if cb, ok := b.Commands[cmd]; ok {
			cb(e)
		}
	} else if strings.HasPrefix(e.Message, nick) {
		// TODO: We may want to ensure at least one space between nick and msg
		// NOTE: We need a copy to not mess up other PRIVMSG handlers
		m := *e
		m.Message = e.Message[len(nick):]
		m.Message = strings.TrimLeftFunc(m.Message, func(r rune) bool {
			// http://weknowgifs.com/wp-content/uploads/2013/03/its-magic-shia-labeouf-gif.gif
			return unicode.IsPunct(r) || unicode.IsSpace(r)
		})
		for _, v := range b.MentionCommands {
			v(&m)
		}
	}
}

func (b *Bot) join(e *irc.Event) {
	b.UserLock.Lock()
	defer b.UserLock.Unlock()

	if e.Nick != b.Conn.GetNick() {
		b.addChannelToNick(e.Message, e.Nick)
	} else {
		// For safety, if we join the channel,
		// make sure we aren't tracking anyone there yet
		for _, user := range b.Users {
			b.removeChannelFromUser(e.Message, user)
		}
	}
}

func (b *Bot) whoReply(e *irc.Event) {
	b.UserLock.Lock()
	defer b.UserLock.Unlock()

	fmt.Println(len(e.Arguments))
	if len(e.Arguments) < 7 {
		// TODO: Error here
		return
	}

	fmt.Printf("%+v\n", e.Arguments)
	b.addChannelToNick(e.Arguments[1], e.Arguments[5])
}

func (b *Bot) nick(e *irc.Event) {
	b.UserLock.Lock()
	defer b.UserLock.Unlock()

	data := b.GetUser(e.Nick)
	if len(data.Channels) == 0 {
		return
	}

	data.CurrentNick = e.Message

	delete(b.Users, e.Nick)
	b.Users[e.Message] = data
}

func (b *Bot) addChannelToNick(channel string, nick string) {
	// NOTE: This assumes the UserLock is already acquired
	user := b.GetUser(nick)

	for i := 0; i < len(user.Channels); i++ {
		if user.Channels[i] == channel {
			return
		}
	}

	user.Channels = append(user.Channels, channel)
	b.Users[nick] = user
}

func (b *Bot) removeChannelFromUser(channel string, user *User) {
	// NOTE: This assumes the UserLock is already acquired
	for i := 0; i < len(user.Channels); i++ {
		if user.Channels[i] == channel {
			// Swap with last element and shrink slice
			user.Channels[i] = user.Channels[len(user.Channels)-1]
			user.Channels = user.Channels[:len(user.Channels)-1]
			break
		}
	}

	if len(user.Channels) == 0 {
		// Removing user
		delete(b.Users, user.CurrentNick)
	}
}

func (b *Bot) part(e *irc.Event) {
	b.UserLock.Lock()
	defer b.UserLock.Unlock()

	if len(e.Arguments) < 1 {
		// TODO: Error here
		return
	}

	if e.Nick != b.Conn.GetNick() {
		if user, ok := b.Users[e.Nick]; ok {
			b.removeChannelFromUser(e.Arguments[0], user)
		}
	} else {
		// At this point, we left, so clean up that chanel
		for _, user := range b.Users {
			b.removeChannelFromUser(e.Arguments[0], user)
		}
	}
}

func (b *Bot) quit(e *irc.Event) {
	b.UserLock.Lock()
	defer b.UserLock.Unlock()

	// TODO: Implement this
}
