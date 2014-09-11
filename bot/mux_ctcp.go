package  bot

import (
	"strings"
	"sync"

	"bitbucket.org/belak/irc"
)

type CTCPMux struct {
	handlers map[string][]BotFunc
	lock     *sync.RWMutex
}

func NewCTCPMux() *CTCPMux {
	mux := &CTCPMux{
		make(map[string][]BotFunc),
		&sync.RWMutex{},
	}

	return mux
}

func (m *CTCPMux) Event(c string, h BotFunc) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.handlers[c] = append(m.handlers[c], h)
}

func (m *CTCPMux) HandleEvent(b *Bot, e *irc.Event) {
	if e.Command != "CTCP" {
		return
	}


	c := strings.SplitN(e.Trailing(), " ", 1)[0]
	newEvent := e.Copy()

	m.lock.RLock()
	defer m.lock.RUnlock()

	handlers, ok := m.handlers[c]
	if !ok {
		// No handlers for this CTCP type
		return
	}

	for _, h := range handlers {
		h(b, newEvent)
	}
}
