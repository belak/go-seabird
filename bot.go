package seabird

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Sirupsen/logrus"

	"github.com/belak/go-plugin"
	"github.com/belak/go-seabird/internal"
	"github.com/belak/irc"
)

type coreConfig struct {
	Nick string
	User string
	Name string
	Pass string

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
}

// NewBot will return a new Bot given the name of a toml config file.
func NewBot(confReader io.Reader) (*Bot, error) {
	var err error

	b := &Bot{
		NewBasicMux(),
		make(map[string]toml.Primitive),
		toml.MetaData{},
		coreConfig{},
		nil,
		plugins.Copy(),
		nil,
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
	if m.FromChannel() {
		target = m.Params[0]
	}

	b.Send(&irc.Message{
		Prefix:  &irc.Prefix{},
		Command: "PRIVMSG",
		Params: []string{
			target,
			fmt.Sprintf(format, v...),
		},
	})

	return nil
}

// MentionReply acts the same as Bot.Reply but it will prefix it with the user's
// nick if we are in a channel.
func (b *Bot) MentionReply(m *irc.Message, format string, v ...interface{}) error {
	if m.FromChannel() {
		format = "%s: " + format
		v = internal.Prepend(v, m.Prefix.Name)
	}

	return b.Reply(m, format, v...)
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

func (b *Bot) handler(c *irc.Client, m *irc.Message) {
	// Handle the event and pass it along
	if m.Command == "001" {
		b.log.Info("Connected")

		for _, v := range b.config.Cmds {
			b.Write(v)
		}
	}

	b.mux.HandleEvent(b, m)
}

// Run starts the bot and loops until it dies
func (b *Bot) Run() error {
	// TODO: We currently ignore the injector, but it could be nice to keep it
	// around for optional plugins.
	_, err := b.registry.Load(b.config.Plugins, nil)
	if err != nil {
		return err
	}

	// The ReadWriteCloser will contain either a *net.Conn or *tls.Conn
	var c io.ReadWriteCloser
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

	// Create a client from the connection we've just opened
	rc := irc.ClientConfig{
		Nick: b.config.Nick,
		Pass: b.config.Pass,
		User: b.config.User,
		Name: b.config.Name,

		Handler: irc.HandlerFunc(b.handler),
	}
	b.client = irc.NewClient(c, rc)

	// Now that we have a client, set up debug callbacks
	b.client.Reader.DebugCallback = func(line string) {
		b.log.Debug("<-- ", strings.Trim(line, "\r\n"))
	}
	b.client.Writer.DebugCallback = func(line string) {
		b.log.Debug("--> ", strings.Trim(line, "\r\n"))
	}

	/* DebugCallback was removed in belak/irc so we should work around
	it at some point.

		b.client.DebugCallback = func(operation, line string) {
			b.log.WithField("op", operation).Debug(line)
			if operation == "write" {
				if len(line) > 512 {
					b.log.WithFields(logrus.Fields{
						"op":  operation,
						"msg": line,
					}).Warn("Output line longer than 512 chars")
				}
				if strings.ContainsAny(line, "\n\r") {
					b.log.WithFields(logrus.Fields{
						"op":  operation,
						"msg": line,
					}).Warn("Output contains a newline")
				}
			}
		}
	*/

	// Start the main loop
	return b.client.Run()
}
