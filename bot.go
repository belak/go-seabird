package seabird

import (
	"github.com/thoj/go-ircevent"
	"labix.org/v2/mgo"

	"crypto/tls"
	"encoding/json"
	"errors"
	"strings"
	"sync"
)

type Config struct {
	Prefix  string
	Verbose bool

	// Bot info
	Nick string
	User string
	Name string
	Pass string

	// Host
	Host        string
	TLS         bool
	TLSNoVerify bool

	// Cmds for on connect
	Cmds []string

	// Plugin config
	AuthPlugin  string
	AuthPlugins map[string]json.RawMessage
	Plugins     map[string]json.RawMessage
}

type User struct {
	CurrentNick string
	Account     string
	Channels    []string
}

type Bot struct {
	// Anything global that we'll need
	Conn *irc.Connection

	// Maps nick to account
	Users    map[string]*User
	UserLock sync.Mutex

	// Any callbacks we're handling
	Commands        map[string]Callback
	MentionCommands []Callback

	// Mongo stuff
	DB *mgo.Database

	// Config
	Config *Config

	// Auth plugin
	Auth AuthPlugin
}

func (b *Bot) GetUser(nick string) *User {
	user, ok := b.Users[nick]
	if !ok {
		user = &User{CurrentNick: nick}
	}

	return user
}

func NewBot(c *Config) (*Bot, error) {
	bot := &Bot{}

	bot.Users = make(map[string]*User)

	bot.Commands = make(map[string]Callback)
	bot.MentionCommands = make([]Callback, 0)

	bot.Config = c
	bot.Conn = irc.IRC(bot.Config.Nick, bot.Config.User)
	bot.Conn.Password = bot.Config.Pass

	bot.Conn.VerboseCallbackHandler = bot.Config.Verbose

	sess, err := mgo.Dial("localhost")
	if err != nil {
		return nil, err
	}

	bot.DB = sess.DB("seabird")

	// Hook it up to all the required functions
	bot.Conn.AddCallback("001", bot.connect)
	bot.Conn.AddCallback("PRIVMSG", bot.msg)
	bot.Conn.AddCallback("JOIN", bot.join)
	bot.Conn.AddCallback("NICK", bot.nick)
	bot.Conn.AddCallback("PART", bot.part)
	bot.Conn.AddCallback("QUIT", bot.quit)

	// Alright, now we're getting to the weird callbacks
	bot.Conn.AddCallback("352", bot.whoReply)

	if ap, ok := auth_plugins[bot.Config.AuthPlugin]; ok && bot.Config.AuthPlugin != "nil" {
		bot.Auth = ap(bot, bot.Config.AuthPlugins[bot.Config.AuthPlugin])
	} else {
		// TODO: Log this
		bot.Auth = NewNilAuthPlugin(bot)
	}

	for k, v := range plugins {
		v(bot, bot.Config.Plugins[k])
	}

	return bot, err
}

func (b *Bot) Loop() {
	b.Conn.UseTLS = b.Config.TLS
	if b.Config.TLSNoVerify {
		b.Conn.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	b.Conn.Connect(b.Config.Host)
	b.Conn.Loop()
}

func (b *Bot) MentionReply(e *irc.Event, format string, v ...interface{}) error {
	// TODO: Confirm it's a privmsg
	// Sanity check
	if len(e.Arguments) == 0 || len(e.Arguments[0]) == 0 {
		return errors.New("invalid irc event")
	}

	if e.Arguments[0][0] == '#' {
		format = strings.Replace(e.Nick, "%", "%%", -1) + ": " + format
	}

	b.Reply(e, format, v...)

	return nil
}

// Reply to a specific event
func (b *Bot) Reply(e *irc.Event, format string, v ...interface{}) error {
	// TODO: Confirm it's a privmsg
	// Sanity check
	if len(e.Arguments) == 0 || len(e.Arguments[0]) == 0 {
		return errors.New("invalid irc event")
	}

	if e.Arguments[0][0] == '#' {
		b.Conn.Privmsgf(e.Arguments[0], format, v...)
	} else {
		b.Conn.Privmsgf(e.Nick, format, v...)
	}

	return nil
}
