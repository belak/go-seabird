// Code generated by "stringer -type ColorCode"; DO NOT EDIT

package uno

import "fmt"

const _ColorCodeName = "ColorNoneColorRedColorYellowColorGreenColorBlue"

var _ColorCodeIndex = [...]uint8{0, 9, 17, 28, 38, 47}

func (i ColorCode) String() string {
	if i < 0 || i >= ColorCode(len(_ColorCodeIndex)-1) {
		return fmt.Sprintf("ColorCode(%d)", i)
	}
	return _ColorCodeName[_ColorCodeIndex[i]:_ColorCodeIndex[i+1]]
}
