package bot

import (
	"strings"

	"bitbucket.org/belak/seabird/irc"
)

type CTCPMux struct {
	handlers *BasicMux
}

func NewCTCPMux() *CTCPMux {
	return &CTCPMux{
		NewBasicMux(),
	}
}

func (m *CTCPMux) Event(c string, h BotFunc) {
	m.handlers.Event(c, h)
}

func (m *CTCPMux) HandleEvent(b *Bot, e *irc.Event) {
	if e.Command != "CTCP" {
		// TODO: Log this
		return
	}

	// Copy it into a new event
	newEvent := e.Copy()

	// Modify the new event
	lastArg := e.Trailing()
	msgParts := strings.SplitN(lastArg, " ", 1)
	newEvent.Args[len(newEvent.Args)-1] = ""
	if len(msgParts) > 1 {
		newEvent.Args[len(newEvent.Args)-1] = msgParts[1]
	}

	newEvent.Command = msgParts[0]

	m.handlers.HandleEvent(b, newEvent)
}
