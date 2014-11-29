package mux

import (
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
}

// This will create an initialized BasicMux with no handlers.
func NewCommandMux(prefix string) *CommandMux {
	return &CommandMux{
		irc.NewBasicMux(),
		irc.NewBasicMux(),
		prefix,
	}
}

// CommandMux.Event will register a Handler
func (m *CommandMux) Event(c string, h irc.HandlerFunc) {
	m.private.Event(c, h)
	m.public.Event(c, h)
}

func (m *CommandMux) Channel(c string, h irc.HandlerFunc) {
	m.public.Event(c, h)
}

func (m *CommandMux) Private(c string, h irc.HandlerFunc) {
	m.private.Event(c, h)
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
