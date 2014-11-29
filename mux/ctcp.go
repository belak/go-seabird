package mux

import (
	"strings"

	"github.com/belak/irc"
)

type CTCPMux struct {
	handlers *irc.BasicMux
}

func NewCTCPMux() *CTCPMux {
	return &CTCPMux{
		irc.NewBasicMux(),
	}
}

func (m *CTCPMux) Event(c string, h irc.HandlerFunc) {
	m.handlers.Event(c, h)
}

func (m *CTCPMux) HandleEvent(c *irc.Client, e *irc.Event) {
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

	m.handlers.HandleEvent(c, newEvent)
}
