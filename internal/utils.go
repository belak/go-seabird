package internal

import (
	"strings"
)

// AppendStr appends string to slice with no duplicates.
func AppendStr(strs []string, str string) []string {
	for _, s := range strs {
		if s == str {
			return strs
		}
	}

	return append(strs, str)
}

// IsSliceContainsStr returns true if the string exists in given slice, ignore
// case.
func IsSliceContainsStr(sl []string, str string) bool {
	str = strings.ToLower(str)

	for _, s := range sl {
		if strings.ToLower(s) == str {
			return true
		}
	}

	return false
}
