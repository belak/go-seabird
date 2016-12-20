package uno

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
)

const handSize = 7

type ColorCode int

const (
	ColorNone ColorCode = iota
	ColorRed
	ColorYellow
	ColorGreen
	ColorBlue
)

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

type cardType int

const (
	cardType0 cardType = iota
	cardType1
	cardType2
	cardType3
	cardType4
	cardType5
	cardType6
	cardType7
	cardType8
	cardType9
	cardTypeSkip
	cardTypeReverse
	cardTypeDrawTwo
	cardTypeWildcard
	cardTypeWildcardDrawFour
)

type GameState int

const (
	StateRunning GameState = iota
	StateWaitingTurn
	StateWaitingColor
	StateWaitingColorFour
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
	for i := cardType1; i < cardTypeWildcard; i++ {
		card := UnoCard{i, color}
		deck.Cards = append(deck.Cards, card)
		deck.Cards = append(deck.Cards, card)
	}
}

func CardTypeString(ct cardType) string {
	switch ct {
	case cardType0:
		return "0"
	case cardType1:
		return "1"
	case cardType2:
		return "2"
	case cardType3:
		return "3"
	case cardType4:
		return "4"
	case cardType5:
		return "5"
	case cardType6:
		return "6"
	case cardType7:
		return "7"
	case cardType8:
		return "8"
	case cardType9:
		return "9"
	case cardTypeSkip:
		return "S"
	case cardTypeReverse:
		return "R"
	case cardTypeDrawTwo:
		return "D"
	case cardTypeWildcard:
		return "W"
	case cardTypeWildcardDrawFour:
		return "W4"
	}
	return ""
}

func (c UnoCard) String() string {
	var color string
	prefix := "\x03"
	switch c.Color {
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
	return fmt.Sprintf("%s[%s]\x030", color, CardTypeString(c.Type))
}

func (c UnoCard) Equals(other UnoCard) bool {
	return c.Color == other.Color && c.Type == other.Type
}

func makeUnoDeck() *unoDeck {
	deck := &unoDeck{}

	addUnoColor(deck, ColorRed)
	addUnoColor(deck, ColorYellow)
	addUnoColor(deck, ColorGreen)
	addUnoColor(deck, ColorBlue)

	wildcard := UnoCard{cardTypeWildcardDrawFour, ColorNone}
	wildcardDrawFour := UnoCard{cardTypeWildcard, ColorNone}
	for i := 0; i < 4; i++ {
		deck.Cards = append(deck.Cards, wildcard)
		deck.Cards = append(deck.Cards, wildcardDrawFour)
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
	hand := make([]UnoCard, handSize)
	for i := 0; i < handSize; i++ {
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
	err := player.DrawCards(g, handSize)
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
	if card.Color != ColorNone {
		if g.nextColor != ColorNone {
			return card.Color == g.nextColor
		}
		return card.Color == topCard.Color
	}

	if card.Type == cardTypeWildcard {
		return true
	}

	for _, other := range player.Hand.Cards {
		if card.Equals(other) {
			continue
		}
		if other.Color != ColorNone && topCard.Color == other.Color {
			return false
		}
	}

	return true
}

func (g *UnoGame) AdvancePlayer() {
	g.playerIndex = (g.playerIndex + g.playDirection) % len(g.Players)
	g.state = StateWaitingTurn
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
	g.nextColor = ColorNone
}

func (g *UnoGame) RunCard(card UnoCard) []string {
	messages := []string{}

	switch card.Type {
	case cardTypeSkip:
		messages = append(messages, fmt.Sprintf("%s skipped!", g.CurrentPlayer().Name))
		g.ClearExpectedColor()
		g.AdvancePlayer()
	case cardTypeDrawTwo:
		g.CurrentPlayer().DrawCards(g, 2)
		messages = append(messages, fmt.Sprintf("%s draws two and skips a turn.", g.CurrentPlayer().Name))
		g.ClearExpectedColor()
		g.AdvancePlayer()
	case cardTypeReverse:
		messages = append(messages, "Play reversed.")
		g.ClearExpectedColor()
		g.Reverse()
		g.AdvancePlayer()
	case cardTypeWildcard:
		messages = append(messages, fmt.Sprintf("%s must declare next color.", g.CurrentPlayer().Name))
		g.state = StateWaitingColor
	case cardTypeWildcardDrawFour:
		messages = append(messages, fmt.Sprintf("%s must declare next color.", g.CurrentPlayer().Name))
		g.state = StateWaitingColorFour
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
	if g.Discard.Top().Type == cardTypeWildcardDrawFour {
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
			}

			messages = append(messages, fmt.Sprintf("Error drawing first card: %s", err))
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
	game := &UnoGame{Deck: makeUnoDeck(), Discard: &unoDeck{}, playerIndex: 0, playDirection: 1, state: StateRunning, nextColor: ColorNone}
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
