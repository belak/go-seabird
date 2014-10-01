package bot

import (
	"strings"

	"bitbucket.org/belak/seabird/irc"
)

// CommandMux is a simple IRC event multiplexer, based on the BasicMux.
//
// The CommandMux is given a prefix string and matches all PRIVMSG
// events which start with it. The first word after the string is
// moved into the Event.Command.
type CommandMux struct {
	private *BasicMux
	public  *BasicMux
	prefix  string
}

// This will create an initialized BasicMux with no handlers.
func NewCommandMux(prefix string) *CommandMux {
	return &CommandMux{
		NewBasicMux(),
		NewBasicMux(),
		prefix,
	}
}

// CommandMux.Event will register a Handler
func (m *CommandMux) Event(c string, h BotFunc) {
	m.private.Event(c, h)
	m.public.Event(c, h)
}

func (m *CommandMux) Channel(c string, h BotFunc) {
	m.public.Event(c, h)
}

func (m *CommandMux) Private(c string, h BotFunc) {
	m.private.Event(c, h)
}

// HandleEvent strips off the prefix, pulls the command out
// and runs HandleEvent on the internal BasicMux
func (m *CommandMux) HandleEvent(b *Bot, e *irc.Event) {
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
		m.public.HandleEvent(b, newEvent)
	} else {
		m.private.HandleEvent(b, newEvent)
	}
}
