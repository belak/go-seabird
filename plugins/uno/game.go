package uno

import (
	"container/ring"

	"github.com/belak/go-seabird/plugins"
)

// Message represents a message to be sent to at least one person.
type Message struct {
	// Target is the user who this message should be sent to. It
	// should be nil if it should be sent to everyone.
	Target *plugins.User

	// Message contains the contents of the message
	Message string

	// Private means that the message should be sent privately, rather
	// than to the channel. This can only be used if Target is
	// non-nil.
	Private bool
}

// Game represents an Uno game.
type Game struct {
	owner *plugins.User

	// We use a circular list here because it represents the current
	// player as well as all the players and can be used to easily
	// switch between turns.
	players *ring.Ring

	deck  []*Card
	state gameState
}

type player struct {
	User *plugins.User
	Hand []*Card
}

// NewGame creates a game, adds the given user as the owner, and as
// the first user in the player ring.
func NewGame(u *plugins.User) (*Game, []*Message) {
	g := &Game{
		owner:   u,
		players: &ring.Ring{Value: &player{User: u}},
	}

	return g, []*Message{
		{
			Message: "Created game",
		},
	}
}

// AddPlayer only works if the game has not been started yet. It will
// add the player to the list of players.
func (g *Game) AddPlayer(u *plugins.User) []*Message {
	if g.state != stateInit {
		return []*Message{{
			Target:  u,
			Message: "You can only join a game which hasn't been started yet.",
		}}
	}

	// We want to add them after the last user, so we take the current
	// user (the first at this point), go to the previous, and add it
	// after that.
	prev := g.players.Prev()
	prev.Link(&player{User: u})

	return []*Message{{
		Target:  u,
		Message: "Welcome to the game!",
	}}
}
