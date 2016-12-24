package uno

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
)

const handSize = 7

// ColorCode is a type for card color codes
type ColorCode int

// NOTE: These don't generate code which passes the linter, so I
// recommend the following:
// sed -i 's/_name/Name/g' *_string.go
// sed -i 's/_index/Index/g' *_string.go
//
//go:generate stringer -type ColorCode
//go:generate stringer -type CardType

// Card color codes
const (
	ColorNone ColorCode = iota
	ColorRed
	ColorYellow
	ColorGreen
	ColorBlue
)

// ColorFromString gets the ColorCode for the given string
func ColorFromString(colorStr string) ColorCode {
	switch colorStr {
	case "red":
		return ColorRed
	case "yellow":
		return ColorYellow
	case "green":
		return ColorGreen
	case "blue":
		return ColorBlue
	default:
		return ColorNone
	}
}

// CardType represents each of the different UNO card types.
type CardType int

// Card types
const (
	CardType0 CardType = iota
	CardType1
	CardType2
	CardType3
	CardType4
	CardType5
	CardType6
	CardType7
	CardType8
	CardType9
	CardTypeSkip
	CardTypeReverse
	CardTypeDrawTwo
	CardTypeWildcard
	CardTypeWildcardDrawFour
)

// GameState is a type for game states
type GameState int

// Various game states
const (
	StateRunning GameState = iota
	StateWaitingTurn
	StateWaitingColor
	StateWaitingColorFour
)

// Card represents an UNO card
type Card struct {
	Type  CardType
	color ColorCode
}

// Deck represents a deck of UNO cards
type Deck struct {
	Cards []Card
}

// Player represents a player in an UNO game
type Player struct {
	Name string
	Hand Deck
}

// Game is a struct containing UNO game state
type Game struct {
	Players       []*Player
	Deck          *Deck
	Discard       *Deck
	playerIndex   int
	playDirection int
	state         GameState
	nextColor     ColorCode
}

func addcolor(deck *Deck, color ColorCode) {
	// This adds 1 of every 0, and two of each other number, skip,
	// reverse, and draw two.
	deck.Cards = append(deck.Cards, Card{0, color})
	for i := CardType1; i <= CardTypeDrawTwo; i++ {
		card := Card{i, color}
		deck.Cards = append(deck.Cards, card)
		deck.Cards = append(deck.Cards, card)
	}
}

func (c Card) String() string {
	var color string
	prefix := "\x03"
	switch c.color {
	case ColorNone:
		color = ""
	case ColorRed:
		color = prefix + "4"
	case ColorYellow:
		color = prefix + "8"
	case ColorGreen:
		color = prefix + "3"
	case ColorBlue:
		color = prefix + "2"
	}
	return fmt.Sprintf("%s[%s]\x030", color, c.Type.String())
}

func (c Card) equals(other Card) bool {
	return c.color == other.color && c.Type == other.Type
}

func newDeck() *Deck {
	deck := &Deck{}

	addcolor(deck, ColorRed)
	addcolor(deck, ColorYellow)
	addcolor(deck, ColorGreen)
	addcolor(deck, ColorBlue)

	wildcard := Card{CardTypeWildcard, ColorNone}
	wildcardDrawFour := Card{CardTypeWildcardDrawFour, ColorNone}
	for i := 0; i < 4; i++ {
		deck.Cards = append(deck.Cards, wildcard)
		deck.Cards = append(deck.Cards, wildcardDrawFour)
	}

	return deck
}

func (d *Deck) String() string {
	var cardStrings []string
	for _, c := range d.Cards {
		cardStrings = append(cardStrings, c.String())
	}

	return strings.Join(cardStrings, "\n")
}

func (d *Deck) shuffle() {
	newCards := make([]Card, len(d.Cards))
	for i, v := range rand.Perm(len(d.Cards)) {
		newCards[v] = d.Cards[i]
	}
	d.Cards = newCards
}

func (d *Deck) add(card Card) {
	d.Cards = append(d.Cards, card)
}

func (d *Deck) addTopFromOther(other *Deck) error {
	card, err := other.draw()
	if err != nil {
		return err
	}
	d.add(card)
	return nil
}

func (d *Deck) empty() bool {
	return len(d.Cards) == 0
}

// Top gets the top card in a deck
func (d *Deck) Top() *Card {
	return &d.Cards[0]
}

func (d *Deck) draw() (Card, error) {
	if d.empty() {
		return Card{}, errors.New("Deck is empty")
	}
	card := d.Cards[0]
	d.Cards = d.Cards[1:]
	return card, nil
}

func (d *Deck) drawHand() (Deck, error) {
	hand := make([]Card, handSize)
	for i := 0; i < handSize; i++ {
		card, err := d.draw()
		if err != nil {
			return Deck{}, err
		}
		hand[i] = card
	}
	deck := Deck{Cards: hand}
	return deck, nil
}

// DrawCards draws a number of cards from the game's deck into the player's hand
func (p *Player) DrawCards(game *Game, number int) error {
	var card Card
	var err error
	for i := 0; i < number; i++ {
		if game.Deck.empty() {
			err = game.shuffleDiscard()
			if err != nil {
				return err
			}
		}
		card, err = game.Deck.draw()
		if err != nil {
			return err
		}
		p.Hand.add(card)
	}
	return nil
}

// RemoveCard removes a card from a player's hand
func (p *Player) RemoveCard(idx int) (Card, error) {
	if idx < 0 || idx >= len(p.Hand.Cards) {
		return Card{}, fmt.Errorf("Bad hand index \"%d\"", idx)
	}
	card := p.Hand.Cards[idx]
	p.Hand.Cards = append(p.Hand.Cards[:idx], p.Hand.Cards[idx+1:]...)
	return card, nil
}

func (g *Game) addPlayer(name string) error {
	player := &Player{Name: name, Hand: Deck{}}
	err := player.DrawCards(g, handSize)
	if err != nil {
		return err
	}
	g.Players = append(g.Players, player)
	return nil
}

// GetPlayer gets a player in a game by name
func (g *Game) GetPlayer(name string) (*Player, error) {
	for _, player := range g.Players {
		if player.Name == name {
			return player, nil
		}
	}
	return nil, fmt.Errorf("Can't find player \"%s\"", name)
}

// CurrentPlayer gets the game's current player
func (g *Game) CurrentPlayer() *Player {
	return g.Players[g.playerIndex]
}

func (g *Game) shuffleDiscard() error {
	topCard, err := g.Discard.draw()
	if err != nil {
		return err
	}

	for !g.Discard.empty() {
		err = g.Deck.addTopFromOther(g.Discard)
		if err != nil {
			return err
		}
	}
	g.Discard.add(topCard)
	g.Deck.shuffle()
	return nil
}

func (g *Game) playable(player *Player, card Card) bool {
	// If it's a wildcard, they can play it no matter what.
	if card.Type == CardTypeWildcard {
		return true
	}

	// If the color or type matches, they're allowed to play it.
	topCard := g.Discard.Top()
	if card.color == g.nextColor || card.Type == topCard.Type {
		return true
	}

	// If we've made it to this point, we need to make sure all the
	// other cards aren't playable because a D4 wild can only be
	// played if the user can't play any other cards.
	if card.Type == CardTypeWildcardDrawFour {
		for _, other := range player.Hand.Cards {
			// We need to skip D4 wilds so we don't recurse forever.
			if card.Type == CardTypeWildcardDrawFour {
				continue
			}

			if g.playable(player, other) {
				return false
			}
		}

		return true
	}

	// This shouldn't be reachable, but it's here just in case.
	return false
}

// AdvancePlayer moves the game to the next player
func (g *Game) AdvancePlayer() {
	g.playerIndex = (g.playerIndex + g.playDirection) % len(g.Players)
	if g.playerIndex < 0 {
		g.playerIndex = len(g.Players) - 1
	}
	g.state = StateWaitingTurn
}

func (g *Game) reverse() {
	g.playDirection *= -1
}

// PlayCard plays a given card
func (g *Game) PlayCard(card Card) []string {
	messages := []string{}

	if !g.playable(g.CurrentPlayer(), card) {
		return append(messages, fmt.Sprintf("%s is not playable right now.", card.String()))
	}

	messages = append(messages, g.runCard(card)...)

	g.AdvancePlayer()
	return messages
}

// DrawCard draws a card from the deck and gives it to the current player
func (g *Game) DrawCard() []string {
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

// State returns the current game state
func (g *Game) State() GameState {
	return g.state
}

func (g *Game) clearExpectedColor() {
	g.nextColor = ColorNone
}

func (g *Game) runCard(card Card) []string {
	messages := []string{}

	switch card.Type {
	case CardTypeSkip:
		messages = append(messages, fmt.Sprintf("%s skipped!", g.CurrentPlayer().Name))
		g.clearExpectedColor()
		g.AdvancePlayer()
	case CardTypeDrawTwo:
		g.CurrentPlayer().DrawCards(g, 2)
		messages = append(messages, fmt.Sprintf("%s draws two and skips a turn.", g.CurrentPlayer().Name))
		g.clearExpectedColor()
		g.AdvancePlayer()
	case CardTypeReverse:
		messages = append(messages, "Play reversed.")
		g.clearExpectedColor()
		g.reverse()
		g.AdvancePlayer()
	case CardTypeWildcard:
		messages = append(messages, fmt.Sprintf("%s must declare next color.", g.CurrentPlayer().Name))
		g.state = StateWaitingColor
	case CardTypeWildcardDrawFour:
		messages = append(messages, fmt.Sprintf("%s must declare next color.", g.CurrentPlayer().Name))
		g.state = StateWaitingColorFour
	default:
		g.clearExpectedColor()
	}
	return messages
}

// FirstTurn runs the game's first turn
func (g *Game) FirstTurn() []string {
	messages := []string{
		fmt.Sprintf("%s's turn.", g.CurrentPlayer().Name),
		fmt.Sprintf("%s is on top of discard.", g.Discard.Top()),
	}
	if g.Discard.Top().Type == CardTypeWildcardDrawFour {
		messages = append(messages, "Wildcard draw four can't be the first card. Let's try again.")

		err := g.Deck.addTopFromOther(g.Discard)
		if err == nil {
			g.Deck.shuffle()

			g.Discard.addTopFromOther(g.Deck)
			// This should really never happen
			if err == nil {
				nextAttemptMessages := g.FirstTurn()
				messages = append(messages, nextAttemptMessages...)
				return messages
			}

			messages = append(messages, fmt.Sprintf("Error drawing first card: %s", err))
		} else {
			// This should really never happen
			messages = append(messages, fmt.Sprintf("Error drawing from discard: %s", err))
		}
	} else {
		messages = append(messages, g.runCard(*g.Discard.Top())...)
	}
	fmt.Println(g.CurrentPlayer().Name)
	messages = append(messages, fmt.Sprintf("%s to play.", g.CurrentPlayer().Name))
	return append(messages, fmt.Sprintf("%s is on top of discard.", g.Discard.Top().String()))
}

// ChooseColor sets the expected color
func (g *Game) ChooseColor(color ColorCode) {
	g.nextColor = color
}

// NewGame constructs and starts a new game
func NewGame(players []string) (*Game, error) {
	game := &Game{
		Deck:          newDeck(),
		Discard:       &Deck{},
		playerIndex:   0,
		playDirection: 1,
		state:         StateRunning,
		nextColor:     ColorNone,
	}
	game.Deck.shuffle()

	var err error
	for _, name := range players {
		err = game.addPlayer(name)
		if err != nil {
			return nil, err
		}
	}

	err = game.Discard.addTopFromOther(game.Deck)
	if err != nil {
		return nil, err
	}

	return game, nil
}
