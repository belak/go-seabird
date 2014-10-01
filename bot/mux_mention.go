package bot

import (
	"strings"
	"sync"
	"unicode"

	"bitbucket.org/belak/seabird/irc"
)

// MentionMux is a simple IRC event multiplexer, based on a slice of Handlers
//
// The MentionMux uses the current Nick and punctuation to determine if the
// Client has been mentioned. The nick, punctuation and any leading or
// trailing spaces are removed from the message.
type MentionMux struct {
	handlers []BotFunc
	lock     *sync.RWMutex
}

// This will create an initialized BasicMux with no handlers.
func NewMentionMux() *MentionMux {
	return &MentionMux{
		nil,
		&sync.RWMutex{},
	}
}

// MentionMux.Event will register a Handler
func (m *MentionMux) Event(h BotFunc) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.handlers = append(m.handlers, h)
}

// HandleEvent strips off the nick punctuation and spaces and runs the handlers
func (m *MentionMux) HandleEvent(b *Bot, e *irc.Event) {
	if e.Command != "PRIVMSG" {
		// TODO: Log this
		return
	}

	lastArg := e.Trailing()
	nick := b.C.CurrentNick()

	// We only handle this event if it starts with the
	// current bot's nick followed by punctuation
	if len(lastArg) < len(nick)+2 ||
		!strings.HasPrefix(lastArg, nick) ||
		!unicode.IsPunct(rune(lastArg[len(nick)])) ||
		lastArg[len(nick)+1] != ' ' {

		return
	}

	// Copy it into a new Event
	newEvent := e.Copy()

	// Strip the nick, punctuation, and spaces from the message
	newEvent.Args[len(newEvent.Args)-1] = strings.TrimSpace(lastArg[len(nick)+1:])

	m.lock.RLock()
	defer m.lock.RUnlock()

	for _, h := range m.handlers {
		h(b, newEvent)
	}
}
