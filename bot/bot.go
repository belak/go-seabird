package bot

// TODO: Help strings and what not

import (
	"crypto/tls"
	"reflect"

	"github.com/Sirupsen/logrus"
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
		logrus.New(),
		make(map[string]Plugin),
		make(map[reflect.Type]reflect.Value),
	}

	{
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

		if level, ok := logLevels[c.LogLevel]; ok {
			b.Log.Level = level
		} else {
			b.Log.WithField("loglevel", c.LogLevel).Error("Log level unknown")
		}
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
		for _, v := range c.Cmds {
			cl.Write(v)
		}
	})

	// Create the actual IRC client
	b.client = irc.NewClient(irc.HandlerFunc(b.basic.HandleEvent), c.Nick, c.User, c.Name, c.Pass)
	b.client.Logger = b.Log
	b.inj.Map(b.client)

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
			b.Log.Errorf("Failure loading plugin '%s': %s", v, err)
			return nil, err
		}
	}

	return b, nil
}

func (b *Bot) Config(name string, c PluginConfig) error {
	return viper.MarshalKey(name, c)
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
