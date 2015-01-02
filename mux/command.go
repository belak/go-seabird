package mux

import (
	"sort"
	"strings"

	"github.com/belak/irc"
)

// CommandMux is a simple IRC event multiplexer, based on the BasicMux.
//
// The CommandMux is given a prefix string and matches all PRIVMSG
// events which start with it. The first word after the string is
// moved into the Event.Command.
type CommandMux struct {
	private *irc.BasicMux
	public  *irc.BasicMux
	prefix  string
	cmdHelp map[string]string
}

// This will create an initialized BasicMux with no handlers.
func NewCommandMux(prefix string) *CommandMux {
	m := &CommandMux{
		irc.NewBasicMux(),
		irc.NewBasicMux(),
		prefix,
		make(map[string]string),
	}
	m.Event("help", "[command]", m.help)
	return m
}

func (m *CommandMux) help(c *irc.Client, e *irc.Event) {
	cmd := strings.TrimSpace(e.Trailing())
	if cmd == "" {
		// Get all keys
		keys := make([]string, 0, len(m.cmdHelp))
		for k := range m.cmdHelp {
			keys = append(keys, k)
		}

		// Sort everything
		sort.Strings(keys)

		if e.FromChannel() {
			// If they said "!help" in a channel, list all available commands
			// TODO: Get the command prefix
			c.Reply(e, "Available commands: %s. Use %shelp [command] for more info.", strings.Join(keys, ", "), m.prefix)
		} else {
			for _, v := range keys {
				c.Reply(e, "%s: %s", v, m.cmdHelp[v])
			}
		}
	} else if desc, ok := m.cmdHelp[cmd]; ok {
		c.Reply(e, "%s: %s", cmd, desc)
	} else {
		c.MentionReply(e, "There is no help available for command %q", cmd)
	}
}

// CommandMux.Event will register a Handler
func (m *CommandMux) Event(c string, d string, h irc.HandlerFunc) {
	m.private.Event(c, h)
	m.public.Event(c, h)

	m.cmdHelp[c] = d
}

func (m *CommandMux) Channel(c string, d string, h irc.HandlerFunc) {
	m.public.Event(c, h)

	m.cmdHelp[c] = d
}

func (m *CommandMux) Private(c string, d string, h irc.HandlerFunc) {
	m.private.Event(c, h)

	m.cmdHelp[c] = d
}

// HandleEvent strips off the prefix, pulls the command out
// and runs HandleEvent on the internal BasicMux
func (m *CommandMux) HandleEvent(c *irc.Client, e *irc.Event) {
	if e.Command != "PRIVMSG" {
		// TODO: Log this
		return
	}

	lastArg := e.Trailing()

	if !strings.HasPrefix(lastArg, m.prefix) {
		return
	}

	// Copy it into a new Event
	newEvent := e.Copy()

	msgParts := strings.SplitN(lastArg, " ", 2)
	newEvent.Args[len(newEvent.Args)-1] = ""
	if len(msgParts) > 1 {
		newEvent.Args[len(newEvent.Args)-1] = msgParts[1]
	}

	newEvent.Command = msgParts[0][len(m.prefix):]

	if newEvent.FromChannel() {
		m.public.HandleEvent(c, newEvent)
	} else {
		m.private.HandleEvent(c, newEvent)
	}
}
