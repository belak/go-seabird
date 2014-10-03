package bot

// TODO: Help strings and what not

import (
	"crypto/tls"
	"errors"
	"fmt"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

	"github.com/belak/seabird/irc"
)

type BotFunc func(b *Bot, e *irc.Event)

type Bot struct {
	C *irc.Client

	DB   *mgo.Database
	sess *mgo.Session

	basic *BasicMux
	cmds  *CommandMux
	ment  *MentionMux
	ctcp  *CTCPMux

	// Simple store of all loaded plugins
	plugins    map[string]Plugin
	authPlugin AuthPlugin

	name string
}

type ClientConfig struct {
	ConnectionName string

	Nick string
	User string
	Name string
	Pass string

	Host        string
	TLS         bool
	TLSNoVerify bool

	Plugins    []string
	AuthPlugin string

	Cmds []string

	Prefix string
}

func NewBot(s *mgo.Session, server string) (*Bot, error) {
	db := s.DB("seabird")

	// Normally we'd use b.GetConfig, but we don't have a Bot object yet
	col := db.C("seabird")
	c := &ClientConfig{}
	err := col.Find(bson.M{"connectionname": server}).One(c)
	if err != nil {
		return nil, err
	}

	// NOTE: We load the client afterwords so we can put in the correct handler
	b := &Bot{
		nil,
		db,
		s,
		NewBasicMux(),
		NewCommandMux(c.Prefix),
		NewMentionMux(),
		NewCTCPMux(),
		make(map[string]Plugin),
		nil,
		server,
	}

	b.basic.Event("PRIVMSG", b.cmds.HandleEvent)
	b.basic.Event("PRIVMSG", b.ment.HandleEvent)
	b.basic.Event("CTCP", b.ctcp.HandleEvent)

	b.C = irc.NewClient(irc.HandlerFunc(b.HandleEvent), c.Nick, c.User, c.Name, c.Pass)

	b.Event("001", func(b *Bot, e *irc.Event) {
		bc := b.GetConfig()
		for _, v := range bc.Cmds {
			b.C.Write(v)
		}
	})

	// Initialize Auth plugin first because other plugins may need it
	pf, ok := authPlugins[c.AuthPlugin]
	if !ok {
		return nil, errors.New(fmt.Sprintf("There is not an auth plugin named '%s'", c.AuthPlugin))
	}

	p, err := pf(b)
	if err != nil {
		return nil, err
	}
	b.authPlugin = p

	for _, v := range c.Plugins {
		pf, ok := plugins[v]
		if !ok {
			return nil, errors.New(fmt.Sprintf("There is not a plugin named '%s'", v))
		}

		p, err := pf(b)
		if err != nil {
			return nil, err
		}
		b.plugins[v] = p
	}

	return b, nil
}

func (b *Bot) CheckPerm(nick string, perm string) bool {
	return b.authPlugin.CheckPerm(nick, perm)
}

func (b *Bot) HandleEvent(c *irc.Client, e *irc.Event) {
	b.basic.HandleEvent(b, e)
}

func (b *Bot) Run() error {
	c := b.GetConfig()
	if c.TLS {
		conf := &tls.Config{
			InsecureSkipVerify: c.TLSNoVerify,
		}
		return b.C.DialTLS(c.Host, conf)
	}

	return b.C.Dial(c.Host)
}

func (b *Bot) GetConfig() *ClientConfig {
	c := &ClientConfig{}
	col := b.DB.C("seabird")
	err := col.Find(bson.M{"connectionname": b.name}).One(c)
	if err != nil {
		return nil
	}

	return c
}

func (b *Bot) Event(name string, h BotFunc) {
	b.basic.Event(name, h)
}

func (b *Bot) CTCP(name string, h BotFunc) {
	b.ctcp.Event(name, h)
}

func (b *Bot) Mention(h BotFunc) {
	b.ment.Event(h)
}

func (b *Bot) Command(name string, help string, h BotFunc) {
	b.cmds.Event(name, h)
}

func (b *Bot) CommandPrivate(name string, help string, h BotFunc) {
	b.cmds.Private(name, h)
}

func (b *Bot) CommandPublic(name string, help string, h BotFunc) {
	b.cmds.Channel(name, h)
}

func (b *Bot) Reply(e *irc.Event, format string, args ...interface{}) {
	b.C.Reply(e, format, args...)
}

func (b *Bot) MentionReply(e *irc.Event, format string, args ...interface{}) {
	b.C.MentionReply(e, format, args...)
}

func (b *Bot) CTCPReply(e *irc.Event, format string, args ...interface{}) {
	b.C.CTCPReply(e, format, args...)
}

func (b *Bot) LoadConfig(name string, config interface{}) error {
	col := b.DB.C("config")
	err := col.Find(bson.M{"pluginname": name}).One(config)
	if err != nil {
		return err
	}

	return nil
}
