package seabird

import (
	"testing"

	"github.com/stretchr/testify/require"

	irc "gopkg.in/irc.v3"
)

type messageHandler struct {
	count int
}

func (mh *messageHandler) Handle(b *Bot, m *irc.Message) {
	mh.count++
}

func TestBasicMux(t *testing.T) {
	m := irc.MustParseMessage("001")
	m2 := irc.MustParseMessage("002")

	// Single message, single handler
	mh := &messageHandler{}
	mux := NewBasicMux()
	mux.Event("001", mh.Handle)
	mux.HandleEvent(nil, m)
	require.Equal(t, 1, mh.count)
	mux.HandleEvent(nil, m)
	require.Equal(t, 2, mh.count)

	// Single message, multiple handlers
	mh = &messageHandler{}
	mh2 := &messageHandler{}
	mux = NewBasicMux()
	mux.Event("001", mh.Handle)
	mux.Event("001", mh2.Handle)
	mux.HandleEvent(nil, m)
	require.Equal(t, 1, mh.count)
	require.Equal(t, 1, mh2.count)
	mux.HandleEvent(nil, m)
	require.Equal(t, 2, mh.count)
	require.Equal(t, 2, mh2.count)

	// Different messages, wildcard handler
	mh = &messageHandler{}
	mux = NewBasicMux()
	mux.Event("*", mh.Handle)
	mux.HandleEvent(nil, m)
	require.Equal(t, 1, mh.count)
	mux.HandleEvent(nil, m2)
	require.Equal(t, 2, mh.count)

	// No handlers
	mh = &messageHandler{}
	mux = NewBasicMux()
	mux.HandleEvent(nil, m)
}
