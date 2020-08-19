package seabird

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/sirupsen/logrus"
	irc "gopkg.in/irc.v3"

	"github.com/belak/go-seabird/internal"
)

//nolint:maligned
type coreConfig struct {
	Nick string
	User string
	Name string
	Pass string

	PingFrequency internal.Duration
	PingTimeout   internal.Duration

	Host        string
	TLS         bool
	TLSNoVerify bool
	TLSCert     string
	TLSKey      string

	Cmds   []string
	Prefix string

	Plugins []string

	Debug    bool
	LogLevel string

	SendLimit internal.Duration
	SendBurst int
}

// A Bot is our wrapper around the irc.Client. It could be used for a general
// client, but the provided convenience functions are designed around using this
// package to write a bot.
type Bot struct {
	mux        *BasicMux
	commandMux *CommandMux
	mentionMux *MentionMux

	// Config stuff
	confValues map[string]toml.Primitive
	md         toml.MetaData
	config     coreConfig

	// Internal things
	client         *irc.Client
	log            *logrus.Entry
	context        context.Context
	loadedPlugins  map[string]bool
	loadingContext []string
}

// NewBot will return a new Bot given an io.Reader pointing to a
// config file.
func NewBot(confReader io.Reader) (*Bot, error) {
	var err error

	b := &Bot{
		mux:           NewBasicMux(),
		confValues:    make(map[string]toml.Primitive),
		md:            toml.MetaData{},
		loadedPlugins: make(map[string]bool),
	}

	// Decode the file, but leave all the config sections intact so we can
	// decode those later.
	b.md, err = toml.DecodeReader(confReader, &b.confValues)
	if err != nil {
		return nil, err
	}

	// Load up the core config
	err = b.Config("core", &b.config)
	if err != nil {
		return nil, err
	}

	// Set up logging/debugging
	b.log = logrus.NewEntry(logrus.New())

	b.log.Logger.Level = logrus.InfoLevel
	if b.config.LogLevel != "" {
		level, innerErr := logrus.ParseLevel(b.config.LogLevel)
		if innerErr != nil {
			return nil, innerErr
		}

		b.log.Logger.Level = level
	} else if b.config.Debug {
		b.log.Warn("The Debug config option has been replaced with LogLevel")
		b.log.Logger.Level = logrus.DebugLevel
	}

	b.commandMux = NewCommandMux(b.config.Prefix)
	b.mentionMux = NewMentionMux()

	b.mux.Event("PRIVMSG", b.commandMux.HandleEvent)
	b.mux.Event("PRIVMSG", b.mentionMux.HandleEvent)

	b.context = withSeabirdValues(context.TODO(), b, b.log)

	return b, nil
}

func (b *Bot) Context() context.Context {
	return b.context
}

func (b *Bot) SetValue(key interface{}, value interface{}) {
	b.context = context.WithValue(b.context, key, value)
}

func (b *Bot) BasicMux() *BasicMux {
	return b.mux
}

func (b *Bot) CommandMux() *CommandMux {
	return b.commandMux
}

func (b *Bot) MentionMux() *MentionMux {
	return b.mentionMux
}

// Config will decode the config section for the given name into the given
// interface{}.
func (b *Bot) Config(name string, c interface{}) error {
	if v, ok := b.confValues[name]; ok {
		return b.md.PrimitiveDecode(v, c)
	}

	return fmt.Errorf("Config section for %q missing", name)
}

func (b *Bot) handler(c *irc.Client, m *irc.Message) {
	r := NewRequest(b.context, b, c.CurrentNick(), m)

	// Handle the event and pass it along
	if r.Message.Command == "001" {
		b.log.Info("Connected")

		for _, v := range b.config.Cmds {
			b.client.Write(v)
		}
	} else if r.Message.Command == "PRIVMSG" {
		// Clean up CTCP stuff so plugins don't need to parse it manually
		lastArg := r.Message.Trailing()
		lastIdx := len(lastArg) - 1
		if lastIdx > 0 && lastArg[0] == '\x01' && lastArg[lastIdx] == '\x01' {
			r.Message.Command = "CTCP"
			r.Message.Params[len(r.Message.Params)-1] = lastArg[1:lastIdx]
		}
	}

	b.mux.HandleEvent(r)
}

// ConnectAndRun is a convenience function which will pull the connection
// information out of the config and connect, then call Run.
func (b *Bot) ConnectAndRun() error {
	// The ReadWriteCloser will contain either a *net.Conn or *tls.Conn
	var (
		c   io.ReadWriteCloser
		err error
	)

	if b.config.TLS {
		conf := &tls.Config{
			InsecureSkipVerify: b.config.TLSNoVerify, //nolint:gosec
		}

		if b.config.TLSCert != "" && b.config.TLSKey != "" {
			var cert tls.Certificate
			cert, err = tls.LoadX509KeyPair(b.config.TLSCert, b.config.TLSKey)

			if err != nil {
				return err
			}

			conf.Certificates = []tls.Certificate{cert}
			conf.BuildNameToCertificate()
		}

		c, err = tls.Dial("tcp", b.config.Host, conf)
	} else {
		c, err = net.Dial("tcp", b.config.Host)
	}

	if err != nil {
		return err
	}

	return b.Run(c)
}

func (b *Bot) EnsurePlugin(name string) error {
	loaded, ok := b.loadedPlugins[name]
	if !ok {
		return fmt.Errorf("Plugin %q not loaded", name)
	}

	// If it's already loaded, return nil
	if loaded {
		return nil
	}

	return b.loadPlugin(name)
}

func (b *Bot) loadPlugin(name string) error {
	tmpLoadingContext := append(b.loadingContext, name)

	if internal.IsSliceContainsStr(b.loadingContext, name) {
		return fmt.Errorf(
			"Plugin load loop: %s",
			strings.Join(tmpLoadingContext, ", "))
	}

	// Push the current plugin onto the stack
	b.loadingContext = tmpLoadingContext

	// Note that this is where it's possible for a plugin to recurse.
	// EnsurePlugin can be called by Plugins which can in turn call loadPlugin.
	err := plugins[name](b)

	// Mark the plugin as loaded
	b.loadedPlugins[name] = true

	// Pop the current plugin off the stack
	b.loadingContext = b.loadingContext[:len(b.loadingContext)-1]

	return err
}

func (b *Bot) loadPlugins() error {
	pluginNames, err := matchingPlugins(b.config.Plugins)
	if err != nil {
		return err
	}

	// Update the loadedPlugins map to say which ones we're loading.
	for _, name := range pluginNames {
		b.loadedPlugins[name] = false
	}

	// Loop through all our plugins and load them
	for _, name := range pluginNames {
		err = b.EnsurePlugin(name)
		if err != nil {
			return err
		}
	}

	return nil
}

// Run starts the bot and loops until it dies. It accepts a ReadWriter. If you
// wish to use the connection feature from the config, use ConnectAndRun.
func (b *Bot) Run(c io.ReadWriteCloser) error {
	err := b.loadPlugins()
	if err != nil {
		return err
	}

	// Create a client from the connection we've just opened
	rc := irc.ClientConfig{
		Nick: b.config.Nick,
		Pass: b.config.Pass,
		User: b.config.User,
		Name: b.config.Name,

		PingFrequency: b.config.PingFrequency.Duration,
		PingTimeout:   b.config.PingTimeout.Duration,

		SendLimit: b.config.SendLimit.Duration,
		SendBurst: b.config.SendBurst,

		Handler: irc.HandlerFunc(b.handler),
	}

	b.client = irc.NewClient(c, rc)

	// Now that we have a client, set up debug callbacks
	b.client.Reader.DebugCallback = func(line string) {
		b.log.Debug("<-- ", strings.Trim(line, "\r\n"))
	}
	b.client.Writer.DebugCallback = func(line string) {
		if len(line) > 512 {
			b.log.Warnf("Line longer than 512 chars: %s", strings.Trim(line, "\r\n"))
		}

		b.log.Debug("--> ", strings.Trim(line, "\r\n"))
	}

	// Start the main loop
	return b.client.Run()
}

// Send is a simple function to send an IRC event.
func (b *Bot) WriteMessage(m *irc.Message) {
	b.client.WriteMessage(m)
}

// Write will write an raw IRC message to the stream.
func (b *Bot) Write(line string) {
	b.client.Write(line)
}

// Writef is a convenience method around fmt.Sprintf and Bot.Write.
func (b *Bot) Writef(format string, args ...interface{}) {
	b.client.Writef(format, args...)
}
