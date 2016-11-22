package seabird

import (
	"testing"

	"github.com/belak/irc"
	"github.com/stretchr/testify/assert"
)

func TestCommandMux(t *testing.T) {
	// Empty mux should still have help
	mux := NewCommandMux("!")
	assert.Equal(t, 1, len(mux.cmdHelp))

	mh := &messageHandler{}

	// Ensure simple commands can be hit
	mux.Event("hello", mh.Handle, nil)
	mux.HandleEvent(nil, irc.ParseMessage(":belak PRIVMSG #hello :!hello"))
	assert.Equal(t, 1, mh.count)
	mux.HandleEvent(nil, irc.ParseMessage(":belak PRIVMSG bot :!hello"))
	assert.Equal(t, 2, mh.count)

	// Ensure private commands don't work publicly
	mux = NewCommandMux("!")
	mh = &messageHandler{}
	mux.Private("hello", mh.Handle, nil)
	mux.HandleEvent(nil, irc.ParseMessage(":belak PRIVMSG #hello :!hello"))
	assert.Equal(t, 0, mh.count)
	mux.HandleEvent(nil, irc.ParseMessage(":belak PRIVMSG bot :!hello"))
	assert.Equal(t, 1, mh.count)

	// Ensure public commands don't work publicly
	mux = NewCommandMux("!")
	mh = &messageHandler{}
	mux.Channel("hello", mh.Handle, nil)
	mux.HandleEvent(nil, irc.ParseMessage(":belak PRIVMSG #hello :!hello"))
	assert.Equal(t, 1, mh.count)
	mux.HandleEvent(nil, irc.ParseMessage(":belak PRIVMSG bot :!hello"))
	assert.Equal(t, 1, mh.count)

	// Ensure commands are separate
	mux = NewCommandMux("!")
	mh = &messageHandler{}
	mh2 := &messageHandler{}
	mux.Event("hello1", mh.Handle, nil)
	mux.Event("hello2", mh2.Handle, nil)
	mux.HandleEvent(nil, irc.ParseMessage(":belak PRIVMSG #hello :!hello1"))
	assert.Equal(t, 1, mh.count)
	assert.Equal(t, 0, mh2.count)
	mux.HandleEvent(nil, irc.ParseMessage(":belak PRIVMSG #hello :!hello2"))
	assert.Equal(t, 1, mh.count)
	assert.Equal(t, 1, mh2.count)
}
