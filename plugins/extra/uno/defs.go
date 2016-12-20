package uno

import (
	"errors"
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
	Players     []unoPlayer
	Deck        *unoDeck
	playerIndex int
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

func (g *UnoGame) addPlayer(name string) error {
	hand, err := g.Deck.DrawHand()
	if err != nil {
		return err
	}
	g.Players = append(g.Players, unoPlayer{Name: name, Hand: hand})
	return nil
}

func NewGame(players []string) (*UnoGame, error) {
	game := &UnoGame{Deck: makeUnoDeck()}
	game.Deck.Shuffle()

	var err error
	for _, name := range players {
		err = game.addPlayer(name)
		if err != nil {
			return nil, err
		}
	}

	return game, nil
}

//func (g *UnoGame)
