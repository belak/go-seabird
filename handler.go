package seabird

import "github.com/go-irc/irc"

// Handler is an interface representing objects which can be registered to serve
// a particular Event.Command or subcommand in the IRC client.
type Handler interface {
	// HandleEvent should read the data, formulate a response action (if needed)
	// and then return. Returning signals that the Handler is done with the
	// current Event and will let the IRC client move on to the next Handler or
	// Event.
	//
	// Note that if there are calls that may block for a long time such as
	// network requests and IO, it may be best to grab the required data and run
	// the response code in a goroutine so the rest of the Client can continue
	// as usual.
	HandleEvent(b *Bot, m *irc.Message)
}

// The HandlerFunc is an adapter to allow the use of ordinary functions as IRC
// handlers. If f is a function with the appropriate signature, HandlerFunc(f)
// is a Handler object that calls f.
type HandlerFunc func(b *Bot, m *irc.Message)

// HandleEvent calls f(c, e)
func (f HandlerFunc) HandleEvent(b *Bot, m *irc.Message) {
	f(b, m)
}
