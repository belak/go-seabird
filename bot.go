package seabird

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/codegangsta/inject"
	"github.com/lrstanley/girc"
	"github.com/sirupsen/logrus"

	plugin "github.com/belak/go-plugin"
)

// Any internal types are provided as constants here.
const MENTION = "SEABIRD_MENTION"

func PrefixCommand(name string) string {
	return "SEABIRD_COMMAND-" + strings.ToUpper(name)
}

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

	Debug    bool
	LogLevel string
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
	// Config stuff
	confValues map[string]toml.Primitive
	md         toml.MetaData
	config     coreConfig

	// Internal things
	client   *girc.Client
	registry *plugin.Registry
	log      *logrus.Entry
	injector inject.Injector
}

// NewBot will return a new Bot given an io.Reader pointing to a
// config file.
func NewBot(confReader io.Reader) (*Bot, error) {
	var err error

	b := &Bot{
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

	b.log.Logger.Level = logrus.InfoLevel
	if b.config.LogLevel != "" {
		level, err := logrus.ParseLevel(b.config.LogLevel)
		if err != nil {
			return nil, err
		}
		b.log.Logger.Level = level
	} else if b.config.Debug {
		b.log.Warn("The Debug config option has been replaced with LogLevel")
		b.log.Logger.Level = logrus.DebugLevel
	}

	// Register all the things we want with the plugin registry.
	b.registry.RegisterProvider("seabird/core", func() *Bot {
		return b
	})

	return b, nil
}

// GetLogger grabs the underlying logger for this bot.
func (b *Bot) GetLogger() *logrus.Entry {
	// TODO: Remove this
	return b.log
}

// Config will decode the config section for the given name into the given
// interface{}
func (b *Bot) Config(name string, c interface{}) error {
	if v, ok := b.confValues[name]; ok {
		return b.md.PrimitiveDecode(v, c)
	}

	return fmt.Errorf("Config section for %q missing", name)
}

// Run starts the bot and loops until it dies. It will pull the connection
// information out of the config and connect, then wait for the connection to
// end.
func (b *Bot) Run() error {
	var err error
	var tlsConf *tls.Config
	if b.config.TLS {
		tlsConf := &tls.Config{
			InsecureSkipVerify: b.config.TLSNoVerify,
		}

		if b.config.TLSCert != "" && b.config.TLSKey != "" {
			var cert tls.Certificate
			cert, err = tls.LoadX509KeyPair(b.config.TLSCert, b.config.TLSKey)
			if err != nil {
				return err
			}

			tlsConf.Certificates = []tls.Certificate{cert}
			tlsConf.BuildNameToCertificate()
		}
	}

	host, portRaw, err := net.SplitHostPort(b.config.Host)
	if err != nil {
		return err
	}

	port, err := strconv.Atoi(portRaw)
	if err != nil {
		return err
	}

	// Create a client from the connection we've just opened
	b.client = girc.New(girc.Config{
		Nick: b.config.Nick,
		User: b.config.User,
		Name: b.config.Name,

		Server:     host,
		Port:       port,
		ServerPass: b.config.Pass,

		SSL:       b.config.TLS,
		TLSConfig: tlsConf,
	})

	registry := b.registry.Copy()
	registry.RegisterProvider("irc/core", func() *girc.Client {
		return b.client
	})

	b.client.Handlers.Add(girc.ALL_EVENTS, func(client *girc.Client, event girc.Event) {
		b.log.Infof("%+v", event)
	})

	b.client.Handlers.Add(girc.RPL_WELCOME, func(client *girc.Client, event girc.Event) {
		client.Cmd.SendRaw(b.config.Cmds...)
	})

	b.client.Handlers.Add(girc.PRIVMSG, func(client *girc.Client, event girc.Event) {
		if !event.IsFromChannel() {
			return
		}

		last := event.Last()
		if !strings.HasPrefix(last, b.config.Prefix) {
			return
		}

		/*
		   -       // Copy it into a new Event
		   -       newEvent := msg.Copy()
		   -
		   -       // Chop off the command itself
		   -       msgParts := strings.SplitN(lastArg, " ", 2)
		   -       newEvent.Params[len(newEvent.Params)-1] = ""
		   -       if len(msgParts) > 1 {
		   -               newEvent.Params[len(newEvent.Params)-1] = strings.TrimSpace(msgParts[1])
		   -       }
		   -
		   -       newEvent.Command = strings.ToLower(msgParts[0])
		   -       if strings.HasPrefix(newEvent.Command, m.prefix) {
		   -               newEvent.Command = newEvent.Command[len(m.prefix):]
		   -       }
		*/

		// Copy it into a new Event
		newEvent := event.Copy()

		// Chop off the command itself
		msgParts := strings.SplitN(last, " ", 2)
		newEvent.Params[len(newEvent.Params)-1] = ""
		if len(msgParts) > 1 {
			newEvent.Params[len(newEvent.Params)-1] = strings.TrimSpace(msgParts[1])
		}

		// Chop off the prefix and set the command to the internal prefix name
		newEvent.Command = PrefixCommand(msgParts[0][len(b.config.Prefix):])

		client.RunHandlers(newEvent)
	})

	b.client.Handlers.Add(girc.PRIVMSG, func(client *girc.Client, event girc.Event) {
		last := event.Last()
		nick := client.GetNick()

		// We only handle this event if it starts with the current bot's nick
		// followed by punctuation
		if len(last) < len(nick)+2 ||
			!strings.HasPrefix(last, nick) ||
			!unicode.IsPunct(rune(last[len(nick)])) ||
			last[len(nick)+1] != ' ' {

			return
		}

		// Create a new event and set the command to "SEABIRD-MENTION"
		newEvent := event.Copy()
		newEvent.Command = MENTION

		// TODO: create a "help" function

		client.RunHandlers(newEvent)
	})

	b.injector, err = registry.Load(b.config.Plugins, nil)
	if err != nil {
		return err
	}

	// Now that we have a client, set up debug callbacks
	/*
		b.client.Reader.DebugCallback = func(line string) {
			b.log.Debug("<-- ", strings.Trim(line, "\r\n"))
		}
		b.client.Writer.DebugCallback = func(line string) {
			if len(line) > 512 {
				b.log.Warnf("Line longer than 512 chars: %s", strings.Trim(line, "\r\n"))
			}
			b.log.Debug("--> ", strings.Trim(line, "\r\n"))
		}
	*/

	// Start the main loop
	return b.client.Connect()
}
