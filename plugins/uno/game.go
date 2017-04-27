package uno

import (
	"container/ring"
	"fmt"
	"math/rand"
	"strings"

	"github.com/belak/go-seabird/plugins"
)

type gameState int

const (
	stateNew gameState = iota
	stateNeedsPlay
	stateNeedsColor
	statePostDraw
	stateDone
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

	announcePlayer bool
	currentColor   ColorCode
	reversed       bool
	deck           []Card
	discard        []Card
	state          gameState
	playedLast     *player
}

type player struct {
	User      *plugins.User
	Hand      []Card
	CalledUno bool
}

// NewGame creates a game, adds the given user as the owner, and as
// the first user in the player ring.
func NewGame(u *plugins.User) (*Game, []*Message) {
	g := &Game{
		owner:          u,
		players:        &ring.Ring{Value: &player{User: u}},
		announcePlayer: true,
	}

	return g, []*Message{{
		Target:  u,
		Message: "Created game",
	}}
}

func (g *Game) announceIfNeeded() []*Message {
	if !g.announcePlayer {
		return nil
	}
	g.announcePlayer = false

	p := g.currentPlayer()

	handStrings := []string{}
	for _, card := range p.Hand {
		handStrings = append(handStrings, card.String())
	}

	return []*Message{
		{
			Message: fmt.Sprintf("It is now %s's turn", g.currentPlayer().User.Nick),
		},
		{
			Target:  p.User,
			Private: true,
			Message: "Your hand: " + strings.Join(handStrings, ", "),
		},
	}
}

func (g *Game) lastPlayed() Card {
	return g.discard[len(g.discard)-1]
}

func (g *Game) advancePlay() {
	// Any time we go to the next player, we need to tell them and
	// send their hand.
	g.announcePlayer = true

	if g.reversed {
		g.players = g.players.Prev()
	} else {
		g.players = g.players.Next()
	}
}

func (g *Game) currentPlayer() *player {
	return g.players.Value.(*player)
}

func (g *Game) nextPlayer() *player {
	if g.reversed {
		return g.players.Prev().Value.(*player)
	}

	return g.players.Next().Value.(*player)
}

func (g *Game) prevPlayer() *player {
	if g.reversed {
		return g.players.Next().Value.(*player)
	}

	return g.players.Prev().Value.(*player)
}

func (g *Game) draw() (Card, bool) {
	var refreshed bool

	if len(g.deck) == 0 {
		refreshed = true

		// Grab the last card because that still needs to be on the
		// discard pile.
		tmp := g.discard[len(g.discard)-1]

		//Add all the cards from discard other than the last one to
		//the deck
		g.deck = g.discard[:len(g.discard)-1]

		// Add the last card back onto the discard pile
		g.discard = []Card{tmp}

		// Shuffle the draw pile
		g.shuffle()
	}

	ret := g.deck[0]
	g.deck = g.deck[1:]
	return ret, refreshed
}

func (g *Game) drawN(n int, target *player) []*Message {
	var msgs []*Message
	var newCards []Card

	msgs = append(msgs, &Message{
		Message: fmt.Sprintf("%s drew %d cards", target.User.Nick, n),
	})

	for i := 0; i < n; i++ {
		card, shuffledDraw := g.draw()
		newCards = append(newCards, card)

		if shuffledDraw {
			msgs = append(msgs, &Message{
				Message: "Deck was shuffled",
			})
		}
	}

	// Add the new cards to the target's hand
	target.Hand = append(target.Hand, newCards...)

	// We need to convert the drawn cards to strings so we can send
	// them to the user.
	var newCardStrings []string
	for _, card := range newCards {
		newCardStrings = append(newCardStrings, card.String())
	}

	msgs = append(msgs, &Message{
		Target:  target.User,
		Private: true,
		Message: "New cards: " + strings.Join(newCardStrings, ", "),
	})

	return msgs
}

func (g *Game) shuffle() {
	newDeck := make([]Card, len(g.deck))
	for i, v := range rand.Perm(len(g.deck)) {
		newDeck[v] = g.deck[i]
	}
	g.deck = newDeck
}

func (g *Game) play(p *player, c Card) []*Message {
	i := -1
	for idx, handCard := range p.Hand {
		if c == handCard {
			i = idx
			break
		}
	}

	// If we didn't find it, something bad happened, but we'll ignore
	// it, because this should never be the case.
	if i >= 0 {
		p.Hand = append(p.Hand[:i], p.Hand[i+1:]...)
	}

	g.discard = append(g.discard, c)

	ret := []*Message{{
		Message: fmt.Sprintf("%s played a %s", p.User.Nick, c.String()),
	}}

	moreMsgs := c.Play(g)
	if len(moreMsgs) > 0 {
		ret = append(ret, moreMsgs...)
	}

	announcement := g.announceIfNeeded()
	if len(announcement) > 0 {
		ret = append(ret, announcement...)
	}

	// Since we know it's playable at this point, we can reset the
	// called uno flag.
	p.CalledUno = false

	return ret

}

// SayUno handles both a user calling Uno for themselves or on other
// people
func (g *Game) SayUno(u *plugins.User) []*Message {
	target := g.playedLast
	if target == nil {
		return []*Message{{
			Target:  u,
			Message: "You can't call uno on the first turn.",
		}}
	}

	if target.CalledUno {
		return []*Message{{
			Target:  u,
			Message: fmt.Sprintf("%s has already called uno.", target.User.Nick),
		}}
	}

	// If the user is the previous user, they're calling for themselves
	if target.User == u {
		if len(target.Hand) != 1 {
			return []*Message{{
				Target:  u,
				Message: "You have more than 1 card left!",
			}}
		}

		target.CalledUno = true
		return []*Message{{
			Message: fmt.Sprintf("%s called uno!", u.Nick),
		}}
	}

	if len(target.Hand) != 1 {
		return []*Message{{
			Target:  u,
			Message: fmt.Sprintf("%s has more than 1 card left!", target.User.Nick),
		}}
	}

	// We set CalledUno to true so it can only be called on them once.
	target.CalledUno = true

	// Start with calling uno
	ret := []*Message{{
		Message: fmt.Sprintf("%s called uno on %s!", u.Nick, target.User.Nick),
	}}

	ret = append(ret, g.drawN(4, target)...)

	return ret
}

// AddPlayer only works if the game has not been started yet. It will
// add the player to the list of players.
func (g *Game) AddPlayer(u *plugins.User) []*Message {
	if g.state != stateNew {
		return []*Message{{
			Target:  u,
			Message: "You can only join a game which hasn't been started yet.",
		}}
	}

	// TODO: Ensure the user isn't in the game already

	// We want to add them after the last user, so we take the current
	// user (the first at this point), go to the previous, and add it
	// after that.
	prev := g.players.Prev()
	prev.Link(&ring.Ring{Value: &player{User: u}})

	return []*Message{{
		Target:  u,
		Message: "Welcome to the game!",
	}}
}

// Start will handle setup and starting a game.
func (g *Game) Start(u *plugins.User) []*Message {
	if g.state != stateNew {
		return []*Message{{
			Target:  u,
			Message: "This game is already started!",
		}}
	} else if g.owner != u {
		return []*Message{{
			Target:  u,
			Message: "Only the game owner can start the game!",
		}}
	}

	// For each color we need to add 1 zero, and 2 of every other card.
	for color := ColorRed; color <= ColorYellow; color++ {
		zero := &SimpleCard{color: color, symbol: "0"}
		g.deck = append(g.deck, zero)

		for i := '1'; i <= '9'; i++ {
			card := &SimpleCard{
				color:  color,
				symbol: string(i),
			}

			g.deck = append(g.deck, card, card)
		}
	}

	// Select a top card before any special cards are in here.
	g.shuffle()

	// Grab the first card and take it from the deck
	g.discard = append(g.discard, g.deck[0])
	g.deck = g.deck[1:]

	g.currentColor = g.discard[0].Color()

	// Add in two of all the special cards.
	for color := ColorRed; color <= ColorYellow; color++ {
		drawTwo := NewDrawTwoCard(color)
		reverse := NewReverseCard(color)
		skip := NewSkipCard(color)

		g.deck = append(
			g.deck,

			drawTwo, drawTwo,
			reverse, reverse,
			skip, skip,
		)
	}

	wild := NewWildCard()
	drawfourwild := NewDrawFourWildCard()

	// Add in the wilds
	for i := 0; i < 4; i++ {
		g.deck = append(g.deck, wild)
		g.deck = append(g.deck, drawfourwild)
	}

	g.shuffle()

	var ret = []*Message{
		{
			Target:  u,
			Message: "The game has started! Good luck!",
		},
		{
			Message: "The top card is a " + g.discard[0].String(),
		},
	}

	curPlayer := g.players
	for i := g.players.Len(); i > 0; i-- {
		actualPlayer := curPlayer.Value.(*player)

		// Note: this returns messages but we ignore them during
		// setup.
		g.drawN(7, actualPlayer)

		handStrings := []string{}
		for _, card := range actualPlayer.Hand {
			handStrings = append(handStrings, card.String())
		}

		ret = append(ret, &Message{
			Target:  actualPlayer.User,
			Private: true,
			Message: "Your hand: " + strings.Join(handStrings, ", "),
		})

		curPlayer = curPlayer.Next()
	}

	g.state = stateNeedsPlay

	ret = append(ret, g.announceIfNeeded()...)

	return ret

}

// Stop will handle cleaning up a game. Only the owner can do this. If
// the owner has left, this can be done by anyone.
func (g *Game) Stop(u *plugins.User) ([]*Message, bool) {
	if g.state == stateNew {
		return []*Message{{
			Target:  u,
			Message: "This game hasn't been started started!",
		}}, false
	} else if g.owner.Nick != "" && g.owner != u {
		return []*Message{{
			Target:  u,
			Message: "Only the game owner can stop the game!",
		}}, false
	}

	return []*Message{{
		Target:  u,
		Message: "Game has been stopped",
	}}, true
}

// GetHand will return the hand for the current user.
func (g *Game) GetHand(u *plugins.User) []*Message {
	if g.state <= stateNew || g.state >= stateDone {
		return []*Message{{
			Target:  u,
			Message: "There isn't a game running!",
		}}
	}

	var targetPlayer *player
	curPlayer := g.players
	for i := g.players.Len(); i > 0; i-- {
		actualPlayer := curPlayer.Value.(*player)
		if actualPlayer.User == u {
			targetPlayer = actualPlayer
			break
		}

		curPlayer = curPlayer.Next()
	}

	var handStrings []string
	for _, card := range targetPlayer.Hand {
		handStrings = append(handStrings, card.String())
	}

	return []*Message{{
		Target:  u,
		Private: true,
		Message: "Here's your hand: " + strings.Join(handStrings, ", "),
	}}
}

// Play will handle a user playing a card. It will return messages
// along with if the game is now over.
func (g *Game) Play(u *plugins.User, card string) ([]*Message, bool) {
	if g.state != stateNeedsPlay {
		return []*Message{{
			Target:  u,
			Message: "You can't do that right now!",
		}}, false
	}

	p := g.currentPlayer()
	if p.User != u {
		return []*Message{{
			Target:  u,
			Message: "It's not your turn!",
		}}, false
	}

	var playedCard Card
	for _, handCard := range p.Hand {
		if handCard.String() == card {
			playedCard = handCard
			break
		}
	}

	if playedCard == nil || !playedCard.Playable(g) {
		return []*Message{{
			Target:  u,
			Message: "You can't play that card right now!",
		}}, false
	}

	// This is the win condition
	if len(p.Hand) == 1 {
		return []*Message{{
			Message: fmt.Sprintf("Nice job! %s won!", u.Nick),
		}}, true
	}

	// Update the last player so we can have a target for "uno" calls.
	g.playedLast = p

	return g.play(p, playedCard), false
}

// Draw makes the given player draw a card.
func (g *Game) Draw(u *plugins.User) []*Message {
	if g.state != stateNeedsPlay {
		return []*Message{{
			Target:  u,
			Message: "You can't do that right now!",
		}}
	}

	p := g.currentPlayer()
	if p.User != u {
		return []*Message{{
			Target:  u,
			Message: "It's not your turn!",
		}}
	}

	// We transition to postDraw and not Play because the user can
	// play this card if they can (and want to).
	g.state = statePostDraw

	return append(g.drawN(1, p), &Message{
		Target:  u,
		Message: "Would you like to play the card you drew?",
	})
}

// DrawPlay can only be called after a draw and mostly matters if
// their card is playable.
func (g *Game) DrawPlay(u *plugins.User, action string) []*Message {
	if g.state != statePostDraw {
		return []*Message{{
			Target:  u,
			Message: "You can't do that right now!",
		}}
	}

	p := g.currentPlayer()
	if p.User != u {
		return []*Message{{
			Target:  u,
			Message: "It's not your turn!",
		}}
	}

	// We set the state to needs play. If a wild is played, it's fine
	// that it's overwritten.
	g.state = stateNeedsPlay

	c := p.Hand[len(p.Hand)-1]
	if action == "no" {
		g.advancePlay()
		return g.announceIfNeeded()
	}

	if !c.Playable(g) {
		g.advancePlay()
		return append([]*Message{{
			Target:  u,
			Message: "That card isn't playable. Sorry.",
		}}, g.announceIfNeeded()...)
	}

	return g.play(p, c)
}

// SetColor is a callback used by the wilds.
func (g *Game) SetColor(u *plugins.User, color string) []*Message {
	if g.state != stateNeedsColor {
		return []*Message{{
			Target:  u,
			Message: "You can't do that right now!",
		}}
	}

	p := g.currentPlayer()
	if p.User != u {
		return []*Message{{
			Target:  u,
			Message: "It's not your turn!",
		}}
	}

	g.currentColor = ColorCodeFromString(color)
	if g.currentColor == ColorNone {
		return []*Message{{
			Target:  u,
			Message: "That's not a valid color!",
		}}
	}

	// If the top card is a ColorNotifier, call the ColorChanged
	// callback.
	top := g.discard[len(g.discard)-1]
	colorNotifier, ok := top.(ColorChangeNotifier)

	ret := []*Message{{
		Message: "The color is now " + g.currentColor.String(),
	}}

	if ok {
		moreMsgs := colorNotifier.ColorChanged(g)
		ret = append(ret, moreMsgs...)
		ret = append(ret, g.announceIfNeeded()...)
	}

	return ret
}
