package utils

import (
	"context"
	"sync"

	irc "gopkg.in/irc.v3"
)

// TestHandler is meant to be inserted as a Handler somewhere to
// capture all messages which are sent.
type TestHandler struct {
	messages []*irc.Message
	lock     sync.Mutex
}

// Handle implements the Handler interface.
func (th *TestHandler) Handle(ctx context.Context, m *irc.Message) {
	th.lock.Lock()
	defer th.lock.Unlock()

	th.messages = append(th.messages, m)
}

// PopMessages will return all the messages that were passed to this
// handler.
func (th *TestHandler) PopMessages() []*irc.Message {
	th.lock.Lock()
	defer th.lock.Unlock()

	ret := th.messages
	th.messages = nil

	return ret
}
