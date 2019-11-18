package seabird

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	irc "gopkg.in/irc.v3"
)

type messageHandler struct {
	count int
}

func (mh *messageHandler) Handle(r *Request) {
	mh.count++
}

func TestBasicMux(t *testing.T) {
	r := NewRequest(context.TODO(), nil, "bot", irc.MustParseMessage("001"))
	r2 := NewRequest(context.TODO(), nil, "bot", irc.MustParseMessage("002"))

	// Single message, single handler
	mh := &messageHandler{}
	mux := NewBasicMux()
	mux.Event("001", mh.Handle)
	mux.HandleEvent(r)
	require.Equal(t, 1, mh.count)
	mux.HandleEvent(r)
	require.Equal(t, 2, mh.count)

	// Single message, multiple handlers
	mh = &messageHandler{}
	mh2 := &messageHandler{}
	mux = NewBasicMux()
	mux.Event("001", mh.Handle)
	mux.Event("001", mh2.Handle)
	mux.HandleEvent(r)
	require.Equal(t, 1, mh.count)
	require.Equal(t, 1, mh2.count)
	mux.HandleEvent(r)
	require.Equal(t, 2, mh.count)
	require.Equal(t, 2, mh2.count)

	// Different messages, wildcard handler
	mh = &messageHandler{}
	mux = NewBasicMux()
	mux.Event("*", mh.Handle)
	mux.HandleEvent(r)
	require.Equal(t, 1, mh.count)
	mux.HandleEvent(r2)
	require.Equal(t, 2, mh.count)

	// No handlers
	// mh = &messageHandler{}
	mux = NewBasicMux()
	mux.HandleEvent(r)
}
