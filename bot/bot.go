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

func MessageFromChannel(m *irc.Message) bool {
	if len(m.Params) == 0 {
		return false
	}

	loc := m.Params[0]
	return len(loc) > 0 && (loc[0] == irc.Channel || loc[0] == irc.Distributed)
}

type BotConfig struct {
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
}

type Bot struct {
	// Everything needed for plugins
	BasicMux   *BasicMux
	CommandMux *CommandMux
	MentionMux *MentionMux

	// Stuff needed for the IRC client
	currentNick string

	// Config stuff
	confValues map[string]toml.Primitive
	md         toml.MetaData

	// Internal things
	conn   *irc.Conn
	config *BotConfig
	err    error
}

func NewBot(conf string) (*Bot, error) {
	b := &Bot{
		NewBasicMux(),
		nil,
		NewMentionMux(),
		"",
		make(map[string]toml.Primitive),
		toml.MetaData{},
		nil,
		&BotConfig{},
		nil,
	}

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

func (b *Bot) CurrentNick() string {
	return b.currentNick
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

func (b *Bot) Send(m *irc.Message) {
	if b.err != nil {
		return
	}

	err := b.conn.Encode(m)
	if err != nil {
		b.err = err
	}
}

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

func (b *Bot) Write(line string) {
	b.Send(irc.ParseMessage(line))
}

func (b *Bot) Writef(format string, args ...interface{}) {
	b.Write(fmt.Sprintf(format, args...))
}

func (b *Bot) Run() error {
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
