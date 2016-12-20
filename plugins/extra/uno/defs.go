package uno

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
)

const HAND_SIZE = 7

type ColorCode int

const (
	COLOR_NONE ColorCode = iota
	COLOR_RED
	COLOR_YELLOW
	COLOR_GREEN
	COLOR_BLUE
)

type cardType int

const (
	CARD_TYPE_0 cardType = iota
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

type GameState int

const (
	STATE_RUNNING GameState = iota
	STATE_WAITING_TURN
	STATE_WAITING_COLOR
	STATE_WAITING_COLOR_FOUR
)

type UnoCard struct {
	Type  cardType
	Color ColorCode
}

type unoDeck struct {
	Cards []UnoCard
}

type UnoPlayer struct {
	Name string
	Hand unoDeck
}

type UnoGame struct {
	Players       []*UnoPlayer
	Deck          *unoDeck
	Discard       *unoDeck
	playerIndex   int
	playDirection int
	state         GameState
	nextColor     ColorCode
}

func addUnoColor(deck *unoDeck, color ColorCode) {
	deck.Cards = append(deck.Cards, UnoCard{0, color})
	for i := CARD_TYPE_1; i < CARD_TYPE_WILDCARD; i++ {
		card := UnoCard{cardType(i), color}
		deck.Cards = append(deck.Cards, card)
		deck.Cards = append(deck.Cards, card)
	}
}

func (c UnoCard) String() string {
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
	case COLOR_NONE:
		color = "none"
	case COLOR_RED:
		color = "red"
	case COLOR_YELLOW:
		color = "yellow"
	case COLOR_GREEN:
		color = "green"
	case COLOR_BLUE:
		color = "blue"
	}

	return "[card " + num + " " + color + "]"
}

func (c UnoCard) Equals(other UnoCard) bool {
	return c.Color == other.Color && c.Type == other.Type
}

func makeUnoDeck() *unoDeck {
	deck := &unoDeck{}

	addUnoColor(deck, COLOR_RED)
	addUnoColor(deck, COLOR_YELLOW)
	addUnoColor(deck, COLOR_GREEN)
	addUnoColor(deck, COLOR_BLUE)

	wildcard := UnoCard{CARD_TYPE_WILDCARD_DRAW_FOUR, COLOR_NONE}
	wildcard_draw_four := UnoCard{CARD_TYPE_WILDCARD, COLOR_NONE}
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
	newCards := make([]UnoCard, len(d.Cards))
	for i, v := range rand.Perm(len(d.Cards)) {
		newCards[v] = d.Cards[i]
	}
	d.Cards = newCards
}

func (d *unoDeck) Add(card UnoCard) {
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

func (d *unoDeck) Top() *UnoCard {
	return &d.Cards[0]
}

func (d *unoDeck) Draw() (UnoCard, error) {
	if d.Empty() {
		return UnoCard{}, errors.New("Deck is empty")
	}
	card := d.Cards[0]
	d.Cards = d.Cards[1:]
	return card, nil
}

func (d *unoDeck) DrawHand() (unoDeck, error) {
	hand := make([]UnoCard, HAND_SIZE)
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

func (p *UnoPlayer) DrawCards(game *UnoGame, number int) error {
	var card UnoCard
	var err error
	for i := 0; i < number; i++ {
		if game.Deck.Empty() {
			err = game.ShuffleDiscard()
			if err != nil {
				return err
			}
		}
		card, err = game.Deck.Draw()
		if err != nil {
			return err
		}
		p.Hand.Add(card)
	}
	return nil
}

func (p *UnoPlayer) RemoveCard(idx int) (UnoCard, error) {
	if idx < 0 || idx >= len(p.Hand.Cards) {
		return UnoCard{}, fmt.Errorf("Bad hand index \"%d\"", idx)
	}
	card := p.Hand.Cards[idx]
	p.Hand.Cards = append(p.Hand.Cards[:idx], p.Hand.Cards[idx+1:]...)
	return card, nil
}

func (g *UnoGame) addPlayer(name string) error {
	player := &UnoPlayer{Name: name, Hand: unoDeck{}}
	err := player.DrawCards(g, HAND_SIZE)
	if err != nil {
		return err
	}
	g.Players = append(g.Players, player)
	return nil
}

func (g *UnoGame) GetPlayer(name string) (*UnoPlayer, error) {
	for _, player := range g.Players {
		if player.Name == name {
			return player, nil
		}
	}
	return nil, fmt.Errorf("Can't find player \"%s\"", name)
}

func (g *UnoGame) CurrentPlayer() *UnoPlayer {
	return g.Players[g.playerIndex]
}

func (g *UnoGame) ShuffleDiscard() error {
	// TODO(jsvana): shuffle discard into deck save for top card
	topCard, err := g.Discard.Draw()
	if err != nil {
		return err
	}

	for !g.Discard.Empty() {
		err = g.Deck.AddTopFromOther(g.Discard)
		if err != nil {
			return err
		}
	}
	g.Discard.Add(topCard)
	g.Deck.Shuffle()
	return nil
}

func (g *UnoGame) Playable(player *UnoPlayer, card UnoCard) bool {
	topCard := g.Discard.Top()
	if card.Color != COLOR_NONE {
		if g.nextColor != COLOR_NONE {
			return card.Color == g.nextColor
		}
		return card.Color == topCard.Color
	}

	if card.Type == CARD_TYPE_WILDCARD {
		return true
	}

	for _, other := range player.Hand.Cards {
		if card.Equals(other) {
			continue
		}
		if other.Color != COLOR_NONE && topCard.Color == other.Color {
			return false
		}
	}

	return true
}

func (g *UnoGame) AdvancePlayer() {
	g.playerIndex = (g.playerIndex + g.playDirection) % len(g.Players)
	g.state = STATE_WAITING_TURN
}

func (g *UnoGame) Reverse() {
	g.playDirection *= -1
}

func (g *UnoGame) PlayCard(card UnoCard) []string {
	messages := []string{}

	if !g.Playable(g.CurrentPlayer(), card) {
		return append(messages, fmt.Sprintf("%s is not playable right now.", card.String()))
	}

	messages = append(messages, g.RunCard(card)...)

	g.AdvancePlayer()
	return messages
}

func (g *UnoGame) DrawCard() []string {
	messages := []string{}

	err := g.CurrentPlayer().DrawCards(g, 1)
	if err != nil {
		messages = append(messages, "Error drawing a card")
		return messages
	}

	messages = append(messages, "%s drew a card.", g.CurrentPlayer().Name)

	g.AdvancePlayer()

	messages = append(messages, fmt.Sprintf("%s's turn.", g.CurrentPlayer().Name))
	return append(messages, fmt.Sprintf("%s is on top of discard.", g.Discard.Top()))
}

func (g *UnoGame) State() GameState {
	return g.state
}

func (g *UnoGame) ClearExpectedColor() {
	g.nextColor = COLOR_NONE
}

func (g *UnoGame) RunCard(card UnoCard) []string {
	messages := []string{}

	switch card.Type {
	case CARD_TYPE_SKIP:
		messages = append(messages, fmt.Sprintf("%s skipped!", g.CurrentPlayer().Name))
		g.ClearExpectedColor()
		g.AdvancePlayer()
	case CARD_TYPE_DRAW_TWO:
		g.CurrentPlayer().DrawCards(g, 2)
		messages = append(messages, fmt.Sprintf("%s draws two and skips a turn.", g.CurrentPlayer().Name))
		g.ClearExpectedColor()
		g.AdvancePlayer()
	case CARD_TYPE_REVERSE:
		messages = append(messages, fmt.Sprintf("Play reversed.", g.CurrentPlayer().Name))
		g.ClearExpectedColor()
		g.Reverse()
		g.AdvancePlayer()
	case CARD_TYPE_WILDCARD:
		messages = append(messages, fmt.Sprintf("%s must declare next color.", g.CurrentPlayer().Name))
		g.state = STATE_WAITING_COLOR
	case CARD_TYPE_WILDCARD_DRAW_FOUR:
		messages = append(messages, fmt.Sprintf("%s must declare next color.", g.CurrentPlayer().Name))
		g.state = STATE_WAITING_COLOR_FOUR
	default:
		g.ClearExpectedColor()
	}
	return messages
}

func (g *UnoGame) FirstTurn() []string {
	messages := []string{
		fmt.Sprintf("%s's turn.", g.CurrentPlayer().Name),
		fmt.Sprintf("%s is on top of discard.", g.Discard.Top()),
	}
	if g.Discard.Top().Type == CARD_TYPE_WILDCARD_DRAW_FOUR {
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
	} else {
		messages = append(messages, g.RunCard(*g.Discard.Top())...)
	}
	fmt.Println(g.CurrentPlayer().Name)
	messages = append(messages, fmt.Sprintf("%s to play.", g.CurrentPlayer().Name))
	return append(messages, fmt.Sprintf("%s is on top of discard.", g.Discard.Top().String()))
}

func (g *UnoGame) ChooseColor(color ColorCode) {
	g.nextColor = color
}

func NewGame(players []string) (*UnoGame, error) {
	game := &UnoGame{Deck: makeUnoDeck(), Discard: &unoDeck{}, playerIndex: 0, playDirection: 1, state: STATE_RUNNING, nextColor: COLOR_NONE}
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
