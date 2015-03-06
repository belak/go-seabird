package bot

// TODO: Help strings and what not

import (
	"crypto/tls"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/Sirupsen/logrus"

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
}

type Bot struct {
	// All the things that we need for plugins
	Client     *irc.Client
	BasicMux   *irc.BasicMux
	CommandMux *mux.CommandMux
	MentionMux *mux.MentionMux
	CTCPMux    *mux.CTCPMux

	// Config and logger, stored for later use
	config *CoreConfig
	Log    *logrus.Logger

	// Config stuff
	confValues map[string]toml.Primitive
	md         toml.MetaData
}

func NewBot(conf string) (*Bot, error) {
	// The IRC client is nil so we can fill in the blank in a bit
	// The command mux is also loaded later
	b := &Bot{
		nil,
		irc.NewBasicMux(),
		nil,
		mux.NewMentionMux(),
		mux.NewCTCPMux(),
		&CoreConfig{},
		logrus.New(),
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
	b.CommandMux = mux.NewCommandMux(b.config.Prefix)

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

	// Hook up the other muxes
	b.BasicMux.Event("PRIVMSG", b.CommandMux.HandleEvent)
	b.BasicMux.Event("PRIVMSG", b.MentionMux.HandleEvent)
	b.BasicMux.Event("CTCP", b.CTCPMux.HandleEvent)

	// Run commands on startup
	b.BasicMux.Event("001", func(cl *irc.Client, e *irc.Event) {
		b.Log.Info("Connected")
		for _, v := range b.config.Cmds {
			cl.Write(v)
		}
	})

	// Create the actual IRC client
	b.Client = irc.NewClient(
		irc.HandlerFunc(b.BasicMux.HandleEvent),
		b.config.Nick,
		b.config.User,
		b.config.Name,
		b.config.Pass,
	)

	// Pass in our logrous logger as the IRC logger
	b.Client.Logger = b.Log

	return b, nil
}

func (b *Bot) RegisterPlugin(p Plugin) error {
	return p.Register(b)
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

		err = b.Client.DialTLS(b.config.Host, conf)
	} else {
		err = b.Client.Dial(b.config.Host)
	}

	if err != nil {
		b.Log.Errorf("Lost connection: %s", err)
	}

	return err
}
