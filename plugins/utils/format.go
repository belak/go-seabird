package utils

import (
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/dustin/go-humanize/english"
	"github.com/google/go-github/github"
	"github.com/spf13/cast"
)

func dateFormat(layout string, v interface{}) (string, error) {
	var t time.Time
	var err error

	if gt, ok := v.(*github.Timestamp); ok {
		t = gt.Time
	} else {
		t, err = cast.ToTimeE(v)
		if err != nil {
			return "", err
		}
	}
	return t.Format(layout), nil
}

func templatePluralize(count int, in interface{}) (string, error) {
	word, err := cast.ToStringE(in)
	if err != nil {
		return "", err
	}

	return Pluralize(count, word), nil
}

// Pluralize attempts to pluralize the given word if count > 1
func Pluralize(count int, word string) string {
	return english.Plural(count, word, "")
}

// PrettifyNumber displays a number with commas
func PrettifyNumber(num int) string {
	return humanize.Comma(int64(num))
}
