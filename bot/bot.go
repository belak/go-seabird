package bot

// TODO: Help strings and what not

import (
	"crypto/tls"
	"fmt"
	"reflect"

	"github.com/BurntSushi/toml"
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/inject"

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

	LogLevel string

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

	// Config and logger, stored for later use
	config *CoreConfig
	Log    *logrus.Logger

	// Simple store of all loaded plugins
	values map[reflect.Type]reflect.Value

	// Config stuff
	confValues map[string]toml.Primitive
	md         toml.MetaData
}

func NewBot(conf string) (*Bot, error) {
	// The IRC client is nil so we can fill in the blank in a bit
	// The command mux is also loaded later
	b := &Bot{
		inject.New(),
		nil,
		irc.NewBasicMux(),
		nil,
		mux.NewMentionMux(),
		mux.NewCTCPMux(),
		&CoreConfig{},
		logrus.New(),
		make(map[reflect.Type]reflect.Value),
		make(map[string]toml.Primitive),
		toml.MetaData{},
	}

	var err error
	b.md, err = toml.DecodeFile(conf, b.confValues)
	if err != nil {
		return nil, err
	}

	// Load up the core config
	err = b.Config("core", b.config)
	if err != nil {
		return nil, err
	}

	// Load up the command mux
	b.cmds = mux.NewCommandMux(b.config.Prefix)

	// Default is warn. This table just translates config values to the logrus Level
	logLevels := map[string]logrus.Level{
		"":      logrus.WarnLevel,
		"debug": logrus.DebugLevel,
		"info":  logrus.InfoLevel,
		"warn":  logrus.WarnLevel,
		"error": logrus.ErrorLevel,
		"fatal": logrus.FatalLevel,
		"panic": logrus.PanicLevel,
	}

	// Set the log level if it's valid
	if level, ok := logLevels[b.config.LogLevel]; ok {
		b.Log.Level = level
	} else {
		b.Log.WithField("loglevel", b.config.LogLevel).Error("Log level unknown")
	}

	// Add the bot and logger to the injection mapper
	b.inj.Map(b)
	b.inj.Map(b.Log)

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
		b.Log.Info("Connected")
		for _, v := range b.config.Cmds {
			cl.Write(v)
		}
	})

	// Create the actual IRC client
	b.client = irc.NewClient(
		irc.HandlerFunc(b.basic.HandleEvent),
		b.config.Nick,
		b.config.User,
		b.config.Name,
		b.config.Pass,
	)

	// Pass in our logrous logger as the IRC logger
	b.client.Logger = b.Log

	// Add the client to the dep injection mappings
	b.inj.Map(b.client)

	// If plugins is nil, just add all of them
	if len(b.config.Plugins) == 0 {
		for k := range plugins {
			b.config.Plugins = append(b.config.Plugins, k)
		}
	}

	loadOrder, err := b.determineLoadOrder()
	if err != nil {
		return nil, err
	}

	b.Log.WithField("load_order", loadOrder).Info("Loading plugins")

	// Load each plugin in the order we just determined
	for _, v := range loadOrder {
		b.Log.WithField("plugin", v).Info("Loading plugin")
		err = b.loadPlugin(v)
		if err != nil {
			b.Log.Errorf("Failure loading plugin %q: %s", v, err)
			return nil, err
		}
	}

	return b, nil
}

func (b *Bot) Config(name string, c interface{}) error {
	if v, ok := b.confValues[name]; ok {
		return b.md.PrimitiveDecode(v, c)
	}
	return fmt.Errorf("Config section for %q missing", name)
}

func (b *Bot) Run() error {
	var err error
	if b.config.TLS {
		conf := &tls.Config{
			InsecureSkipVerify: b.config.TLSNoVerify,
		}

		err = b.client.DialTLS(b.config.Host, conf)
	} else {
		err = b.client.Dial(b.config.Host)
	}

	if err != nil {
		b.Log.Errorf("Lost connection: %s", err)
	}

	return err
}
