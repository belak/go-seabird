package uno

import "fmt"

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
	Color ColorCode
}

// Equals is a convenience method for checking if two cards are equal
func (c *Card) Equals(other *Card) bool {
	return c.Color == other.Color && c.Type == other.Type
}

// String implements the Stringer interface
func (c Card) String() string {
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
	return fmt.Sprintf("%s[%s]\x030", color, c.Type.String())
}
