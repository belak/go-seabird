package seabird

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/inject"

	"github.com/belak/go-plugin"
	"github.com/go-irc/irc"
)

type coreConfig struct {
	Nick string
	User string
	Name string
	Pass string

	PingFrequency duration
	PingTimeout   duration

	Host        string
	TLS         bool
	TLSNoVerify bool
	TLSCert     string
	TLSKey      string

	Cmds   []string
	Prefix string

	Plugins []string

	Debug bool
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

// A Bot is our wrapper around the irc.Client. It could be used for a general
// client, but the provided convenience functions are designed around using this
// package to write a bot.
type Bot struct {
	mux *BasicMux

	// Config stuff
	confValues map[string]toml.Primitive
	md         toml.MetaData
	config     coreConfig

	// Internal things
	client   *irc.Client
	registry *plugin.Registry
	log      *logrus.Entry
	injector inject.Injector
}

// NewBot will return a new Bot given an io.Reader pointing to a
// config file.
func NewBot(confReader io.Reader) (*Bot, error) {
	var err error

	b := &Bot{
		mux:        NewBasicMux(),
		confValues: make(map[string]toml.Primitive),
		md:         toml.MetaData{},
		registry:   plugins.Copy(),
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

	if b.config.Debug {
		b.log.Logger.Level = logrus.DebugLevel
	} else {
		b.log.Logger.Level = logrus.InfoLevel
	}

	commandMux := NewCommandMux(b.config.Prefix)
	mentionMux := NewMentionMux()

	b.mux.Event("PRIVMSG", commandMux.HandleEvent)
	b.mux.Event("PRIVMSG", mentionMux.HandleEvent)

	// Register all the things we want with the plugin registry.
	b.registry.RegisterProvider(func() (*Bot, *BasicMux, *CommandMux, *MentionMux) {
		return b, b.mux, commandMux, mentionMux
	})

	return b, nil
}

// GetLogger grabs the underlying logger for this bot.
func (b *Bot) GetLogger() *logrus.Entry {
	return b.log
}

// CurrentNick returns the current nick of the bot.
func (b *Bot) CurrentNick() string {
	return b.client.CurrentNick()
}

// Config will decode the config section for the given name into the given
// interface{}
func (b *Bot) Config(name string, c interface{}) error {
	if v, ok := b.confValues[name]; ok {
		return b.md.PrimitiveDecode(v, c)
	}

	return fmt.Errorf("Config section for %q missing", name)
}

// Send is a simple function to send an IRC event
func (b *Bot) Send(m *irc.Message) {
	b.client.WriteMessage(m)
}

// Reply to an irc.Message with a convenience wrapper around fmt.Sprintf
func (b *Bot) Reply(m *irc.Message, format string, v ...interface{}) error {
	if len(m.Params) < 1 || len(m.Params[0]) < 1 {
		return errors.New("Invalid IRC message")
	}

	target := m.Prefix.Name
	if b.FromChannel(m) {
		target = m.Params[0]
	}

	fullMsg := fmt.Sprintf(format, v...)
	for _, resp := range strings.Split(fullMsg, "\n") {
		b.Send(&irc.Message{
			Prefix:  &irc.Prefix{},
			Command: "PRIVMSG",
			Params: []string{
				target,
				resp,
			},
		})
	}

	return nil
}

// MentionReply acts the same as Bot.Reply but it will prefix it with the user's
// nick if we are in a channel.
func (b *Bot) MentionReply(m *irc.Message, format string, v ...interface{}) error {
	if len(m.Params) < 1 || len(m.Params[0]) < 1 {
		return errors.New("Invalid IRC message")
	}

	target := m.Prefix.Name
	prefix := ""
	if b.FromChannel(m) {
		target = m.Params[0]
		prefix = m.Prefix.Name + ": "
	}

	fullMsg := fmt.Sprintf(format, v...)
	for _, resp := range strings.Split(fullMsg, "\n") {
		b.Send(&irc.Message{
			Prefix:  &irc.Prefix{},
			Command: "PRIVMSG",
			Params: []string{
				target,
				prefix + resp,
			},
		})
	}

	return nil
}

// PrivateReply is similar to Reply, but it will always send privately.
func (b *Bot) PrivateReply(m *irc.Message, format string, v ...interface{}) {
	b.Send(&irc.Message{
		Prefix:  &irc.Prefix{},
		Command: "PRIVMSG",
		Params: []string{
			m.Prefix.Name,
			fmt.Sprintf(format, v...),
		},
	})
}

// CTCPReply is a convenience function to respond to CTCP requests.
func (b *Bot) CTCPReply(m *irc.Message, format string, v ...interface{}) error {
	if m.Command != "CTCP" {
		return errors.New("Invalid CTCP message")
	}

	b.Send(&irc.Message{
		Prefix:  &irc.Prefix{},
		Command: "NOTICE",
		Params: []string{
			m.Prefix.Name,
			fmt.Sprintf(format, v...),
		},
	})

	return nil
}

func (b *Bot) handshake() {
	b.client.Writef("CAP END")
	b.client.Writef("NICK %s", b.config.Nick)
	b.client.Writef("USER %s 0.0.0.0 0.0.0.0 :%s", b.config.User, b.config.Name)
}

// Write will write an raw IRC message to the stream
func (b *Bot) Write(line string) {
	b.client.Write(line)
}

// Writef is a convenience method around fmt.Sprintf and Bot.Write
func (b *Bot) Writef(format string, args ...interface{}) {
	b.client.Writef(format, args...)
}

// FromChannel is a wrapper around the irc package's FromChannel.
func (b *Bot) FromChannel(m *irc.Message) bool {
	return b.client.FromChannel(m)
}

func (b *Bot) handler(c *irc.Client, m *irc.Message) {
	// Handle the event and pass it along
	if m.Command == "001" {
		b.log.Info("Connected")

		for _, v := range b.config.Cmds {
			b.Write(v)
		}
	} else if m.Command == "PRIVMSG" {
		// Clean up CTCP stuff so plugins don't need to parse it manually
		lastArg := m.Trailing()
		lastIdx := len(lastArg) - 1
		if lastIdx > 0 && lastArg[0] == '\x01' && lastArg[lastIdx] == '\x01' {
			m.Command = "CTCP"
			m.Params[len(m.Params)-1] = lastArg[1:lastIdx]
		}
	}

	b.mux.HandleEvent(b, m)
}

// ConnectAndRun is a convenience function which will pull the
// connection information out of the config and connect, then call
// Run.
func (b *Bot) ConnectAndRun() error {
	// The ReadWriteCloser will contain either a *net.Conn or *tls.Conn
	var c io.ReadWriteCloser
	var err error
	if b.config.TLS {
		conf := &tls.Config{
			InsecureSkipVerify: b.config.TLSNoVerify,
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

// Run starts the bot and loops until it dies. It accepts a
// ReadWriter. If you wish to use the connection feature from the
// config, use ConnectAndRun.
func (b *Bot) Run(c io.ReadWriter) error {
	var err error

	b.injector, err = b.registry.Load(b.config.Plugins, nil)
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
