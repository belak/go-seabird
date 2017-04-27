package uno

import "fmt"

// ColorCode represents the color of a card.
type ColorCode int

const (
	ColorNone ColorCode = iota
	ColorWild
	ColorRed
	ColorGreen
	ColorBlue
	ColorYellow
)

func ColorCodeFromString(color string) ColorCode {
	switch color {
	case "red":
		return ColorRed
	case "blue":
		return ColorBlue
	case "green":
		return ColorGreen
	case "yellow":
		return ColorYellow
	case "wild":
		return ColorWild
	}

	return ColorNone
}

func (cc ColorCode) String() string {
	switch cc {
	case ColorRed:
		return "red"
	case ColorGreen:
		return "green"
	case ColorBlue:
		return "blue"
	case ColorYellow:
		return "yellow"
	case ColorWild:
		return "wild"
	}

	return "unknown"
}

// Card is a generic interface for all cards.
type Card interface {
	// Playable returns true if the card can be played right now and
	// false if it can't. It assumes that the game knows that this is
	// in the hand of the current player.
	Playable(*Game) bool

	// Play applies the effects of this card. It assumes Playable has
	// already been checked. It returns messages explaining additional
	// actions which happened. The plugin will handle basic "[user]
	// played a [card]" and "It is now [user]'s turn" messages.
	Play(*Game) []*Message

	// Color returns the color of this card.
	Color() ColorCode

	// Symbol representing this card.
	Symbol() string

	// String returns how this card should be displayed to players.
	String() string
}

// ColorChangeNotifier is meant to add onto the Card interface. It
// defines what happens when a color is set. This is needed for the
// Wild cards so they can advance the turn after a color is set.
type ColorChangeNotifier interface {
	ColorChanged(*Game) []*Message
}

// BasicCard represents a 0-9
type SimpleCard struct {
	color  ColorCode
	symbol string
}

// Playable implements (Card).Playable
func (c *SimpleCard) Playable(g *Game) bool {
	last := g.lastPlayed()
	return last.Symbol() == c.Symbol() || g.currentColor == c.Color()
}

// Play implements (Card).Play
func (c *SimpleCard) Play(g *Game) []*Message {
	g.currentColor = c.Color()
	g.advancePlay()
	return nil
}

func (c *SimpleCard) Symbol() string {
	return c.symbol
}

func (c *SimpleCard) Color() ColorCode {
	return c.color
}

func (c *SimpleCard) String() string {
	return c.Color().String() + " " + c.Symbol()
}

// DrawTwoCard represents a draw two
type DrawTwoCard struct {
	SimpleCard
}

func NewDrawTwoCard(color ColorCode) *DrawTwoCard {
	return &DrawTwoCard{
		SimpleCard: SimpleCard{
			color:  color,
			symbol: "draw two",
		},
	}
}

// Play implements (Card).Play
func (c *DrawTwoCard) Play(g *Game) []*Message {
	// Move to the next player, draw two cards, then move on
	g.advancePlay()
	target := g.currentPlayer()
	ret := g.drawN(2, target)

	c.SimpleCard.Play(g)

	return ret
}

// SkipCard represents a skip
type SkipCard struct {
	SimpleCard
}

func NewSkipCard(color ColorCode) *SkipCard {
	return &SkipCard{
		SimpleCard: SimpleCard{
			color:  color,
			symbol: "skip",
		},
	}
}

// Play implements (Card).Play
func (c *SkipCard) Play(g *Game) []*Message {
	g.advancePlay()

	ret := []*Message{{
		Message: fmt.Sprintf("%s was skipped.", g.currentPlayer().User.Nick),
	}}

	c.SimpleCard.Play(g)

	return ret
}

// ReverseCard represents a reverse
type ReverseCard struct {
	SimpleCard
}

func NewReverseCard(color ColorCode) *ReverseCard {
	return &ReverseCard{
		SimpleCard: SimpleCard{
			color:  color,
			symbol: "reverse",
		},
	}
}

// Play implements (Card).Play
func (c *ReverseCard) Play(g *Game) []*Message {
	g.reversed = !g.reversed

	c.SimpleCard.Play(g)

	return []*Message{{
		Message: "Play direction has reversed!",
	}}
}

// WildCard represents a wild
type WildCard struct {
	SimpleCard
}

func NewWildCard() *WildCard {
	return &WildCard{
		SimpleCard: SimpleCard{
			color:  ColorWild,
			symbol: "wild",
		},
	}
}

// Playable implements (Card).Playable. This overrides the embedded
// (SimpleCard).Playable method.
func (c *WildCard) Playable(g *Game) bool {
	return true
}

// Play implements (Card).Play. This overrides the embedded
// (SimpleCard).Play method.
func (c *WildCard) Play(g *Game) []*Message {
	g.state = stateNeedsColor
	return []*Message{{
		Target:  g.currentPlayer().User,
		Message: "What color?",
	}}
}

// ColorChanged implements (ColorChangeNotifier).ColorChanged
func (c *WildCard) ColorChanged(g *Game) []*Message {
	g.state = stateNeedsPlay
	g.advancePlay()
	return nil
}

func (c *WildCard) String() string {
	return c.Symbol()
}

// DrawFourWildCard represents a draw four wild.
type DrawFourWildCard struct {
	WildCard
}

func NewDrawFourWildCard() *DrawFourWildCard {
	ret := &DrawFourWildCard{
		WildCard: *NewWildCard(),
	}
	ret.symbol = "draw four wild"

	return ret
}

// Playable implements (Card).Playable. This overrides the embedded
// (WildCard).Playable method.
func (c *DrawFourWildCard) Playable(g *Game) bool {
	p := g.currentPlayer()
	for _, rawHandCard := range p.Hand {
		if rawHandCard.Color() == g.currentColor {
			return false
		}
	}
	return true
}

// ColorChanged implements (ColorChangeNotifier).Color. This overrides
// the embedded (WildCard).ColorChanged method.
func (c *DrawFourWildCard) ColorChanged(g *Game) []*Message {
	g.advancePlay()
	target := g.currentPlayer()
	ret := g.drawN(4, target)

	c.WildCard.Play(g)

	return ret
}
