package bot

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/nightlyone/lockfile"

	"github.com/belak/irc"
)

type coreConfig struct {
	PidFile string

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

	Quiet bool
}

// A Bot is our wrapper around the irc.Client. It could be used for a
// general client, but the provided convenience functions are designed
// around using this package to write a bot.
type Bot struct {
	// Everything needed for plugins
	BasicMux   *BasicMux
	CommandMux *CommandMux
	MentionMux *MentionMux

	Auth AuthProvider

	Plugins map[string]interface{}

	// Config stuff
	confValues map[string]toml.Primitive
	md         toml.MetaData

	// Internal things
	conn                *irc.Conn
	connected           bool
	currentNick         string
	config              *coreConfig
	err                 error
	initialCapList      map[string]bool // This is a map so we avoid dupes
	initialCapResponses int
	loadingPlugins      map[string]bool
}

// NewBot will return a new Bot given the name of a toml config file.
func NewBot(conf string) (*Bot, error) {
	b := &Bot{
		NewBasicMux(),
		nil,
		NewMentionMux(),
		&nullAuthProvider{},
		make(map[string]interface{}),
		make(map[string]toml.Primitive),
		toml.MetaData{},
		nil,
		false,
		"",
		&coreConfig{},
		nil,
		make(map[string]bool),
		0,
		make(map[string]bool),
	}

	// Decode the file, but leave all the config sections intact
	// so we can decode those later.
	b.md, b.err = toml.DecodeFile(conf, b.confValues)
	if b.err != nil {
		return nil, b.err
	}

	// Load up the core config
	b.err = b.Config("core", b.config)
	if b.err != nil {
		return nil, b.err
	}

	// Load up the command mux
	b.CommandMux = NewCommandMux(b.config.Prefix)

	// Hook up the other muxes
	b.BasicMux.Event("PRIVMSG", b.CommandMux.HandleEvent)
	b.BasicMux.Event("PRIVMSG", b.MentionMux.HandleEvent)

	return b, nil
}

func (b *Bot) CapReq(caps ...string) {
	if b.connected {
		// If we're already connected, we don't care about
		// negotiation. We just want to ask for it.
		b.Send(&irc.Message{
			Prefix:  &irc.Prefix{},
			Command: "CAP",
			Params: []string{
				"REQ",
				strings.Join(caps, " "),
			},
		})
	} else {
		// TODO: This is not technically correct. We should be
		// handling requests which start with a -
		for _, cap := range caps {
			b.initialCapList[cap] = true
		}
	}
}

// CurrentNick returns the current nick of the bot.
func (b *Bot) CurrentNick() string {
	return b.currentNick
}

// Config will decode the config section for the given name into the
// given interface{}
func (b *Bot) Config(name string, c interface{}) error {
	if v, ok := b.confValues[name]; ok {
		return b.md.PrimitiveDecode(v, c)
	}
	return fmt.Errorf("Config section for %q missing", name)
}

// Send is a simple function to send an IRC event
func (b *Bot) Send(m *irc.Message) {
	if b.err != nil {
		return
	}

	b.conn.WriteMessage(m)
}

// Reply to an irc.Message with a convenience wrapper around
// fmt.Sprintf
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

// MentionReply acts the same as Bot.Reply but it will prefix it with
// the user's nick if we are in a channel.
func (b *Bot) MentionReply(m *irc.Message, format string, v ...interface{}) error {
	if m.FromChannel() {
		format = "%s: " + format
		v = prepend(v, m.Prefix.Name)
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

// HasPerm is a convenience function for checking a user's permissions
func (b *Bot) HasPerm(m *irc.Message, perm string) bool {
	user := b.Auth.LookupUser(b, m.Prefix)

	return user.HasPerm(b, perm)
}

func (b *Bot) handshake() {
	b.conn.Writef("CAP END")
	b.conn.Writef("NICK %s", b.config.Nick)
	b.conn.Writef("USER %s 0.0.0.0 0.0.0.0 :%s", b.config.User, b.config.Name)
}

func (b *Bot) mainLoop() error {
	// Send the initial connection info
	if len(b.config.Pass) > 0 {
		b.conn.Writef("PASS %s", b.config.Pass)
	}

	// Set the nick setting as the current nick
	b.currentNick = b.config.Nick

	// If we have any capabilities, we need to send requests for
	// them. Note that we send multiple requests because it makes
	// parsing the ACK and NAK commands a lot simpler.
	if len(b.initialCapList) > 0 {
		var caps []string
		for cap := range b.initialCapList {
			caps = append(caps, cap)
			b.Send(&irc.Message{
				Prefix:  &irc.Prefix{},
				Command: "CAP",
				Params:  []string{"REQ", cap},
			})
		}
	} else {
		// We can only handshake now if we didn't have any
		// caps we wanted responses to.
		b.handshake()
	}

	var m *irc.Message
	for {
		m, b.err = b.conn.ReadMessage()
		if b.err != nil {
			break
		}

		// Internal handlers to make sure we track the
		// currentNick correctly and send PONGs
		if m.Command == "001" {
			log.Println("Connected")
			b.connected = true

			for _, v := range b.config.Cmds {
				b.Write(v)
			}

		} else if m.Command == "CAP" {
			if len(m.Params) > 0 {
				if m.Params[0] == "ACK" {
					// Because we send each CAP
					// individually, we shouldn't
					// need to do anything here.

					b.initialCapResponses++
				} else if m.Params[0] == "NAK" {
					return fmt.Errorf("Got CAP NAK for %s", m.Params[1])
				}
			}

			if b.initialCapResponses >= len(b.initialCapList) {
				// Now that we've got all the
				// responses back that we needed, we
				// can continue the initial dance.
				b.handshake()
			}
		} else if m.Command == "NICK" {
			if m.Prefix.Name == b.currentNick && len(m.Params) > 0 {
				b.currentNick = m.Params[0]
			}
		} else if m.Command == "PING" {
			b.conn.Writef("PONG :%s", m.Trailing())
		} else if m.Command == "001" {
			b.currentNick = m.Params[0]
		} else if m.Command == "437" || m.Command == "433" {
			b.currentNick = b.currentNick + "_"
			b.conn.Writef("NICK %s", b.currentNick)
		}

		b.BasicMux.HandleEvent(b, m)

		// TODO: Make this work better
		if b.err != nil {
			break
		}
	}

	return b.err
}

// Write will write an raw IRC message to the stream
func (b *Bot) Write(line string) {
	b.conn.Write(line)
}

// Writef is a convenience method around fmt.Sprintf and Bot.Write
func (b *Bot) Writef(format string, args ...interface{}) {
	b.conn.Writef(format, args...)
}

// PluginLoaded will return true if a plugin is loaded and false otherwise
func (b *Bot) PluginLoaded(name string) bool {
	_, ok := b.Plugins[name]
	return ok
}

// LoadPlugin will ensure a plugin is loaded. It is designed to be
// usable in other plugins, so they can ensure plugins they depend on
// are loaded before using them.
func (b *Bot) LoadPlugin(name string) error {
	// We don't need to load the plugin if it's already loaded
	if b.PluginLoaded(name) {
		return nil
	}

	// Ensure the plugin exists
	factory, ok := plugins[name]
	if !ok {
		return fmt.Errorf("Plugin %s does not exist", name)
	}

	// If we're trying to load this plugin already, this is a
	// circular load and we should bail.
	if v := b.loadingPlugins[name]; v {
		return fmt.Errorf("Plugin %s getting loaded circularly", name)
	}

	// Set a flag stating that we're loading this plugin
	b.loadingPlugins[name] = true

	// Actually load the plugin
	plugin, err := factory(b)
	if err != nil {
		return fmt.Errorf("Plugin %s failed to load", name)
	}

	// Save the value returned from the plugin factory.
	b.Plugins[name] = plugin

	// Unset the flag saying we're loading this because we're done now.
	delete(b.loadingPlugins, name)

	return nil
}

// Run starts the bot and loops until it dies
func (b *Bot) Run() error {
	var err error
	// Load all the plugins we need. If there were plugins
	// specified in the config, just load those. Otherwise load
	// ALL of them.
	if len(b.config.Plugins) != 0 {
		for _, name := range b.config.Plugins {
			err = b.LoadPlugin(name)
			if err != nil {
				return err
			}
		}
	} else {
		for name := range plugins {
			err = b.LoadPlugin(name)
			if err != nil {
				return err
			}
		}
	}

	// If we have a pidfile configured, create it and write the PID
	if b.config.PidFile != "" {
		l, err := lockfile.New(b.config.PidFile)
		if err != nil {
			return err
		}

		err = l.TryLock()
		if err != nil {
			return err
		}

		defer l.Unlock()
	}

	// The ReadWriteCloser will contain either a *net.Conn or *tls.Conn
	var c io.ReadWriteCloser
	if b.config.TLS {
		conf := &tls.Config{
			InsecureSkipVerify: b.config.TLSNoVerify,
		}

		c, err = tls.Dial("tcp", b.config.Host, conf)
	} else {
		c, err = net.Dial("tcp", b.config.Host)
	}

	if err != nil {
		return err
	}

	// Create a client from the connection we've just opened
	b.conn = irc.NewConn(c)

	b.conn.DebugCallback = func(line string) {
		if !b.config.Quiet {
			log.Println(line)
		}
	}

	// Start the main loop
	return b.mainLoop()
}
