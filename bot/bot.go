package bot

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/nightlyone/lockfile"

	"github.com/belak/sorcix-irc"
)

// MessageFromChannel runs a simple check to see if a message came
// from a channel or a person. It is only designed to work on PRIVMSG
// lines.
func MessageFromChannel(m *irc.Message) bool {
	if len(m.Params) == 0 {
		return false
	}

	loc := m.Params[0]
	return len(loc) > 0 && (loc[0] == irc.Channel || loc[0] == irc.Distributed)
}

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
}

// A Bot is our wrapper around the irc.Conn. It could be used for a
// general client, but the provided convenience functions are designed
// around using this package to write a bot.
type Bot struct {
	// Everything needed for plugins
	BasicMux   *BasicMux
	CommandMux *CommandMux
	MentionMux *MentionMux

	Auth AuthProvider

	Plugins map[string]interface{}

	// Stuff needed for the IRC client
	currentNick string

	// Config stuff
	confValues map[string]toml.Primitive
	md         toml.MetaData

	// Internal things
	conn           *irc.Conn
	config         *coreConfig
	err            error
	loadingPlugins map[string]bool
}

// NewBot will return a new Bot given the name of a toml config file.
func NewBot(conf string) (*Bot, error) {
	b := &Bot{
		NewBasicMux(),
		nil,
		NewMentionMux(),
		&nullAuthProvider{},
		make(map[string]interface{}),
		"",
		make(map[string]toml.Primitive),
		toml.MetaData{},
		nil,
		&coreConfig{},
		nil,
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

	b.currentNick = b.config.Nick

	// Load up the command mux
	b.CommandMux = NewCommandMux(b.config.Prefix)

	// Hook up the other muxes
	b.BasicMux.Event("PRIVMSG", b.CommandMux.HandleEvent)
	b.BasicMux.Event("PRIVMSG", b.MentionMux.HandleEvent)

	// Run commands on startup
	b.BasicMux.Event("001", func(bot *Bot, m *irc.Message) {
		log.Println("Connected")
		for _, v := range bot.config.Cmds {
			bot.Write(v)
		}
	})

	return b, nil
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

// Send is a simple function to send an IRC message
func (b *Bot) Send(m *irc.Message) {
	if b.err != nil {
		return
	}

	err := b.conn.Encode(m)
	if err != nil {
		b.err = err
	}
}

// Reply to an irc.Message with a convenience wrapper around
// fmt.Sprintf
func (b *Bot) Reply(m *irc.Message, format string, v ...interface{}) {
	if len(m.Params) == 0 || len(m.Params[0]) == 0 {
		log.Println("Invalid IRC event")
		return
	}

	// Create the base message
	out := &irc.Message{
		Command: "PRIVMSG",
	}

	// Make sure we send it to the right place
	if MessageFromChannel(m) {
		out.Params = append(out.Params, m.Params[0])
	} else {
		out.Params = append(out.Params, m.Prefix.Name)
	}

	// Append the outgoing text
	out.Params = append(out.Params, fmt.Sprintf(format, v...))

	b.Send(out)
}

// MentionReply acts the same as Bot.Reply but it will prefix it with the
// user's nick if we are in a channel.
func (b *Bot) MentionReply(m *irc.Message, format string, v ...interface{}) {
	if len(m.Params) == 0 || len(m.Params[0]) == 0 {
		log.Println("Invalid IRC event")
		return
	}

	if MessageFromChannel(m) {
		format = "%s: " + format
		v = prepend(v, m.Prefix.Name)
	}

	b.Reply(m, format, v...)
}

// CheckPerm is a convenience function for checking a user's permissions
func (b *Bot) HasPerm(m *irc.Message, perm string) bool {
	user := b.Auth.LookupUser(b, m.Prefix)
	
	return user.HasPerm(b, perm)
}

func (b *Bot) mainLoop(conn io.ReadWriteCloser) error {
	b.conn = irc.NewConn(conn)

	// Startup commands
	if len(b.config.Pass) > 0 {
		b.Send(&irc.Message{
			Command: "PASS",
			Params:  []string{b.config.Pass},
		})
	}

	b.Send(&irc.Message{
		Command: "NICK",
		Params:  []string{b.currentNick},
	})

	b.Send(&irc.Message{
		Command: "USER",
		Params:  []string{b.config.User, "0.0.0.0", "0.0.0.0", b.config.Name},
	})

	var m *irc.Message
	for {
		m, b.err = b.conn.Decode()
		if b.err != nil {
			break
		}

		if m.Command == "PING" {
			log.Println("Sending PONG")
			b.Send(&irc.Message{
				Command: "PONG",
				Params:  []string{m.Trailing()},
			})
		} else if m.Command == "PONG" {
			ns, _ := strconv.ParseInt(m.Trailing(), 10, 64)
			delta := time.Duration(time.Now().UnixNano() - ns)

			log.Println("!!! Lag:", delta)
		} else if m.Command == "NICK" {
			if m.Prefix.Name == b.currentNick && len(m.Params) > 0 {
				b.currentNick = m.Params[0]
			}
		} else if m.Command == "001" {
			if len(m.Params) > 0 {
				b.currentNick = m.Params[0]
			}
		} else if m.Command == "437" || m.Command == "433" {
			b.currentNick = b.currentNick + "_"
			b.Send(&irc.Message{
				Command: "NICK",
				Params:  []string{b.currentNick},
			})
		}

		log.Println(m)

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
	b.Send(irc.ParseMessage(line))
}

// Writef is a convenience method around fmt.Sprintf and Bot.Write
func (b *Bot) Writef(format string, args ...interface{}) {
	b.Write(fmt.Sprintf(format, args...))
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
	// Load all the plugins we need. If there were plugins
	// specified in the config, just load those. Otherwise load
	// ALL of them.
	if len(b.config.Plugins) != 0 {
		for _, name := range b.config.Plugins {
			err := b.LoadPlugin(name)
			if err != nil {
				return err
			}
		}
	} else {
		for name := range plugins {
			err := b.LoadPlugin(name)
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

	if b.config.TLS {
		conf := &tls.Config{
			InsecureSkipVerify: b.config.TLSNoVerify,
		}

		tcpConn, err := tls.Dial("tcp", b.config.Host, conf)
		if err != nil {
			return err
		}

		return b.mainLoop(tcpConn)
	}

	tcpConn, err := net.Dial("tcp", b.config.Host)
	if err != nil {
		return err
	}

	return b.mainLoop(tcpConn)
}
