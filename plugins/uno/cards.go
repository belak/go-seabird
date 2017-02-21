package uno

import "fmt"

// TODO: Unexport this
type colorCode int

const (
	colorNone colorCode = iota
	colorRed
	colorGreen
	colorBlue
	colorYellow
)

func colorCodeFromString(color string) colorCode {
	switch color {
	case "red":
		return colorRed
	case "blue":
		return colorBlue
	case "green":
		return colorGreen
	case "yellow":
		return colorYellow
	}

	return colorNone
}

func (cc colorCode) String() string {
	switch cc {
	case colorRed:
		return "red"
	case colorGreen:
		return "green"
	case colorBlue:
		return "blue"
	case colorYellow:
		return "yellow"
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

	// String returns how this card should be displayed to players.
	String() string
}

// ColorChangeNotifier is meant to add onto the Card interface. It
// defines what happens when a color is set. This is needed for the
// Wild cards so they can advance the turn after a color is set.
type ColorChangeNotifier interface {
	ColorChanged(*Game) []*Message
}

type BasicCard struct {
	Color colorCode
	Type  string
}

func (c *BasicCard) Playable(g *Game) bool {
	last, ok := g.lastPlayed().(*BasicCard)
	if ok && last.Type == c.Type {
		return true
	}

	return g.currentColor == c.Color
}

func (c *BasicCard) Play(g *Game) []*Message {
	g.currentColor = c.Color
	g.advancePlay()
	return nil
}

func (c *BasicCard) String() string {
	return c.Color.String() + " " + c.Type
}

type DrawTwoCard struct {
	Color colorCode
}

func (c *DrawTwoCard) Playable(g *Game) bool {
	_, ok := g.lastPlayed().(*DrawTwoCard)
	return ok || g.currentColor == c.Color
}

func (c *DrawTwoCard) Play(g *Game) []*Message {
	g.currentColor = c.Color
	g.advancePlay()

	// Move to the next player, draw two cards, then move on
	target := g.currentPlayer()
	ret := g.drawN(2, target)
	g.advancePlay()

	return ret
}

func (c *DrawTwoCard) String() string {
	return c.Color.String() + " draw two"
}

type SkipCard struct {
	Color colorCode
}

func (c *SkipCard) Playable(g *Game) bool {
	_, ok := g.lastPlayed().(*SkipCard)
	return ok || g.currentColor == c.Color
}

func (c *SkipCard) Play(g *Game) []*Message {
	g.currentColor = c.Color
	g.advancePlay()

	ret := []*Message{{
		Message: fmt.Sprintf("%s was skipped.", g.currentPlayer().User.Nick),
	}}

	g.advancePlay()

	return ret
}

func (c *SkipCard) String() string {
	return c.Color.String() + " skip"
}

type ReverseCard struct {
	Color colorCode
}

func (c *ReverseCard) Playable(g *Game) bool {
	_, ok := g.lastPlayed().(*ReverseCard)
	return ok || g.currentColor == c.Color
}

func (c *ReverseCard) Play(g *Game) []*Message {
	g.currentColor = c.Color
	g.reversed = !g.reversed
	g.advancePlay()

	return []*Message{{
		Message: "Play direction has reversed!",
	}}
}

func (c *ReverseCard) String() string {
	return c.Color.String() + " reverse"
}

type WildCard struct{}

func (c *WildCard) Playable(g *Game) bool {
	return true
}

func (c *WildCard) Play(g *Game) []*Message {
	g.state = stateNeedsColor
	return []*Message{{
		Target:  g.currentPlayer().User,
		Message: "What color?",
	}}
}

func (c *WildCard) ColorChanged(g *Game) []*Message {
	g.state = stateNeedsPlay
	g.advancePlay()

	return nil
}

func (c *WildCard) String() string {
	return "wild"
}

type DrawFourWildCard struct{}

func (c *DrawFourWildCard) Playable(g *Game) bool {
	p := g.currentPlayer()
	for _, rawHandCard := range p.Hand {
		_, ok := rawHandCard.(*DrawFourWildCard)
		if ok {
			continue
		}

		if rawHandCard.Playable(g) {
			return false
		}
	}
	return true
}

func (c *DrawFourWildCard) Play(g *Game) []*Message {
	g.state = stateNeedsColor
	return []*Message{{
		Target:  g.currentPlayer().User,
		Message: "What color?",
	}}
}

func (c *DrawFourWildCard) ColorChanged(g *Game) []*Message {
	g.state = stateNeedsPlay
	g.advancePlay()

	target := g.currentPlayer()
	ret := g.drawN(4, target)
	g.advancePlay()

	return ret
}

func (c *DrawFourWildCard) String() string {
	return "draw four wild"
}
