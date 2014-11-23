package bot

// TODO: Help strings and what not

import (
	"crypto/tls"
	"reflect"

	"github.com/codegangsta/inject"
	"github.com/spf13/viper"

	"github.com/belak/irc"
	"github.com/belak/seabird/mux"
)

type CoreConfig struct {
	Nick string
	User string
	Name string
	Pass string

	Host        string
	TLS         bool
	TLSNoVerify bool

	Cmds   []string
	Prefix string

	Plugins []string
}

type Bot struct {
	// Dep injection
	inj inject.Injector

	// All the things that we need for plugins
	client *irc.Client
	basic  *irc.BasicMux
	cmds   *mux.CommandMux
	ment   *mux.MentionMux
	ctcp   *mux.CTCPMux

	// Config, stored for later use
	config *CoreConfig

	// Simple store of all loaded plugins
	plugins map[string]Plugin
	values  map[reflect.Type]reflect.Value
}

func NewBot() (*Bot, error) {
	c := &CoreConfig{}
	err := viper.MarshalKey("core", c)
	if err != nil {
		return nil, err
	}

	// The IRC client is nil so we can fill in the blank in a bit
	b := &Bot{
		inject.New(),
		nil,
		irc.NewBasicMux(),
		mux.NewCommandMux(c.Prefix),
		mux.NewMentionMux(),
		mux.NewCTCPMux(),
		c,
		make(map[string]Plugin),
		make(map[reflect.Type]reflect.Value),
	}

	b.inj.Map(b)

	// Hook up the other muxes
	b.basic.Event("PRIVMSG", b.cmds.HandleEvent)
	b.basic.Event("PRIVMSG", b.ment.HandleEvent)
	b.basic.Event("CTCP", b.ctcp.HandleEvent)
	b.inj.Map(b.basic)
	b.inj.Map(b.cmds)
	b.inj.Map(b.ment)
	b.inj.Map(b.ctcp)

	// Run commands on startup
	b.basic.Event("001", func(cl *irc.Client, e *irc.Event) {
		for _, v := range c.Cmds {
			cl.Write(v)
		}
	})

	// Create the actual IRC client
	b.client = irc.NewClient(irc.HandlerFunc(b.basic.HandleEvent), c.Nick, c.User, c.Name, c.Pass)
	b.inj.Map(b.client)

	loadOrder, err := b.determineLoadOrder()
	if err != nil {
		return nil, err
	}

	for _, v := range loadOrder {
		err = b.loadPlugin(v)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (b *Bot) Run() error {
	if b.config.TLS {
		conf := &tls.Config{
			InsecureSkipVerify: b.config.TLSNoVerify,
		}
		return b.client.DialTLS(b.config.Host, conf)
	}

	return b.client.Dial(b.config.Host)
}
