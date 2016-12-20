package uno

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
)

const HAND_SIZE = 7

type colorCode int

const (
	CARD_NONE = iota
	CARD_RED
	CARD_YELLOW
	CARD_GREEN
	CARD_BLUE
)

type cardType int

const (
	CARD_TYPE_0 = iota
	CARD_TYPE_1
	CARD_TYPE_2
	CARD_TYPE_3
	CARD_TYPE_4
	CARD_TYPE_5
	CARD_TYPE_6
	CARD_TYPE_7
	CARD_TYPE_8
	CARD_TYPE_9
	CARD_TYPE_SKIP
	CARD_TYPE_REVERSE
	CARD_TYPE_DRAW_TWO
	CARD_TYPE_WILDCARD
	CARD_TYPE_WILDCARD_DRAW_FOUR
)

type unoCard struct {
	Type  cardType
	Color colorCode
}

type unoDeck struct {
	Cards []unoCard
}

type unoPlayer struct {
	Name string
	Hand unoDeck
}

type UnoGame struct {
	Players       []*unoPlayer
	Deck          *unoDeck
	Discard       *unoDeck
	playerIndex   int
	playDirection int
}

func addUnoColor(deck *unoDeck, color colorCode) {
	deck.Cards = append(deck.Cards, unoCard{0, color})
	for i := CARD_TYPE_1; i < CARD_TYPE_WILDCARD; i++ {
		card := unoCard{cardType(i), color}
		deck.Cards = append(deck.Cards, card)
		deck.Cards = append(deck.Cards, card)
	}
}

func (c unoCard) String() string {
	var num string
	switch c.Type {
	case CARD_TYPE_0:
		num = "0"
	case CARD_TYPE_1:
		num = "1"
	case CARD_TYPE_2:
		num = "2"
	case CARD_TYPE_3:
		num = "3"
	case CARD_TYPE_4:
		num = "4"
	case CARD_TYPE_5:
		num = "5"
	case CARD_TYPE_6:
		num = "6"
	case CARD_TYPE_7:
		num = "7"
	case CARD_TYPE_8:
		num = "8"
	case CARD_TYPE_9:
		num = "9"
	case CARD_TYPE_SKIP:
		num = "skip"
	case CARD_TYPE_REVERSE:
		num = "reverse"
	case CARD_TYPE_DRAW_TWO:
		num = "draw_two"
	case CARD_TYPE_WILDCARD:
		num = "wildcard"
	case CARD_TYPE_WILDCARD_DRAW_FOUR:
		num = "wildcard_draw_four"
	}

	var color string
	switch c.Color {
	case CARD_NONE:
		color = "none"
	case CARD_RED:
		color = "red"
	case CARD_YELLOW:
		color = "yellow"
	case CARD_GREEN:
		color = "green"
	case CARD_BLUE:
		color = "blue"
	}

	return "[card " + num + " " + color + "]"
}

func makeUnoDeck() *unoDeck {
	deck := &unoDeck{}

	addUnoColor(deck, CARD_RED)
	addUnoColor(deck, CARD_YELLOW)
	addUnoColor(deck, CARD_GREEN)
	addUnoColor(deck, CARD_BLUE)

	wildcard := unoCard{CARD_TYPE_WILDCARD_DRAW_FOUR, CARD_NONE}
	wildcard_draw_four := unoCard{CARD_TYPE_WILDCARD, CARD_NONE}
	for i := 0; i < 4; i++ {
		deck.Cards = append(deck.Cards, wildcard)
		deck.Cards = append(deck.Cards, wildcard_draw_four)
	}

	return deck
}

func (d *unoDeck) String() string {
	cardStrings := make([]string, len(d.Cards))
	for i, c := range d.Cards {
		cardStrings[i] = c.String()
	}

	return strings.Join(cardStrings, "\n")
}

func (d *unoDeck) Shuffle() {
	newCards := make([]unoCard, len(d.Cards))
	for i, v := range rand.Perm(len(d.Cards)) {
		newCards[v] = d.Cards[i]
	}
	d.Cards = newCards
}

func (d *unoDeck) Add(card unoCard) {
	d.Cards = append(d.Cards, card)
}

func (d *unoDeck) AddTopFromOther(other *unoDeck) error {
	card, err := other.Draw()
	if err != nil {
		return err
	}
	d.Add(card)
	return nil
}

func (d *unoDeck) Empty() bool {
	return len(d.Cards) == 0
}

func (d *unoDeck) Top() *unoCard {
	return &d.Cards[0]
}

func (d *unoDeck) Draw() (unoCard, error) {
	if d.Empty() {
		return unoCard{}, errors.New("Deck is empty")
	}
	card := d.Cards[0]
	d.Cards = d.Cards[1:]
	return card, nil
}

func (d *unoDeck) DrawHand() (unoDeck, error) {
	hand := make([]unoCard, HAND_SIZE)
	for i := 0; i < HAND_SIZE; i++ {
		card, err := d.Draw()
		if err != nil {
			return unoDeck{}, err
		}
		hand[i] = card
	}
	deck := unoDeck{Cards: hand}
	return deck, nil
}

func (p *unoPlayer) DrawCards(deck *unoDeck, number int) error {
	for i := 0; i < HAND_SIZE; i++ {
		card, err := deck.Draw()
		if err != nil {
			return err
		}
		p.Hand.Add(card)
	}
	return nil
}

func (g *UnoGame) addPlayer(name string) error {
	player := &unoPlayer{Name: name, Hand: unoDeck{}}
	err := player.DrawCards(g.Deck, HAND_SIZE)
	if err != nil {
		return err
	}
	g.Players = append(g.Players, player)
	return nil
}

func (g *UnoGame) GetPlayer(name string) (*unoPlayer, error) {
	for _, player := range g.Players {
		if player.Name == name {
			return player, nil
		}
	}
	return nil, fmt.Errorf("Can't find player \"%s\"", name)
}

func (g *UnoGame) CurrentPlayer() *unoPlayer {
	return g.Players[g.playerIndex]
}

func (g *UnoGame) AdvancePlayer() {
	g.playerIndex = (g.playerIndex + g.playDirection) % len(g.Players)
}

func (g *UnoGame) Reverse() {
	g.playDirection *= -1
}

func (g *UnoGame) TakeTurn(card unoCard) {
}

func (g *UnoGame) FirstTurn() []string {
	messages := []string{
		fmt.Sprintf("%s is on top of discard.", g.Discard.Top()),
		fmt.Sprintf("%s's turn.", g.CurrentPlayer().Name),
	}
	switch g.Discard.Top().Type {
	case CARD_TYPE_SKIP:
		messages = append(messages, fmt.Sprintf("%s skipped!", g.CurrentPlayer().Name))
		g.AdvancePlayer()
	case CARD_TYPE_DRAW_TWO:
		g.CurrentPlayer().DrawCards(g.Deck, 2)
		messages = append(messages, fmt.Sprintf("%s draws two and skips a turn.", g.CurrentPlayer().Name))
		g.AdvancePlayer()
	case CARD_TYPE_REVERSE:
		messages = append(messages, fmt.Sprintf("Play reversed.", g.CurrentPlayer().Name))
		g.Reverse()
		g.AdvancePlayer()
	case CARD_TYPE_WILDCARD:
		messages = append(messages, fmt.Sprintf("%s must declare next color.", g.CurrentPlayer().Name))
		// TODO(jsvana): somehow handle input states
	case CARD_TYPE_WILDCARD_DRAW_FOUR:
		messages = append(messages, "Wildcard draw four can't be the first card. Let's try again.")

		err := g.Deck.AddTopFromOther(g.Discard)
		if err == nil {
			g.Deck.Shuffle()

			g.Discard.AddTopFromOther(g.Deck)
			// This should really never happen
			if err == nil {
				nextAttemptMessages := g.FirstTurn()
				messages = append(messages, nextAttemptMessages...)
				return messages
			} else {
				messages = append(messages, fmt.Sprintf("Error drawing first card: %s", err))
			}
		} else {
			// This should really never happen
			messages = append(messages, fmt.Sprintf("Error drawing from discard: %s", err))
		}
	}
	messages = append(messages, fmt.Sprintf("%s to play.", g.CurrentPlayer().Name))
	messages = append(messages, fmt.Sprintf("%s is on top of discard.", g.Discard.Top().String()))

	return messages
}

func NewGame(players []string) (*UnoGame, error) {
	game := &UnoGame{Deck: makeUnoDeck(), Discard: &unoDeck{}, playerIndex: 0, playDirection: 1}
	game.Deck.Shuffle()

	var err error
	for _, name := range players {
		err = game.addPlayer(name)
		if err != nil {
			return nil, err
		}
	}

	err = game.Discard.AddTopFromOther(game.Deck)
	if err != nil {
		return nil, err
	}

	return game, nil
}
