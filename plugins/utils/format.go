package utils

import (
	"math"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/dustin/go-humanize/english"
	"github.com/google/go-github/github"
	"github.com/spf13/cast"
)

func dateFormat(layout string, v interface{}) (string, error) {
	var (
		t   time.Time
		err error
	)

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

// Pluralize attempts to pluralize the given word (and display the number) if
// count > 1
func Pluralize(count int, word string) string {
	return english.Plural(count, word, "")
}

func templatePluralizeWord(count int, in interface{}) (string, error) {
	word, err := cast.ToStringE(in)
	if err != nil {
		return "", err
	}

	return PluralizeWord(count, word), nil
}

// PluralizeWord attempts to pluralize the given word if count > 1
func PluralizeWord(count int, word string) string {
	return english.PluralWord(count, word, "")
}

// PrettifyNumber displays a number with commas
func PrettifyNumber(num int) string {
	return humanize.Comma(int64(num))
}

var defaultSuffixes = []string{"B", "M", "K"}

func templatePrettifySuffix(num int) (string, error) {
	return PrettifySuffix(num), nil
}

// PrettifySuffix displays a semi-human-readable format such as 4k in place of
// 4125.
func PrettifySuffix(num int) string {
	return RawPrettifySuffix(float64(num), 1000, nil)
}

// RawPrettifySuffix displays a semi-human-readable format such as 4k in place of
// 4125 with a number of weird options.
func RawPrettifySuffix(num, blockSize float64, suffixes []string) string {
	if suffixes == nil {
		suffixes = defaultSuffixes
	}

	threshold := math.Pow(blockSize, float64(len(suffixes)))

	for _, suffix := range suffixes {
		if num >= threshold {
			return humanize.FormatFloat("#,###.#", num/threshold) + suffix
		}

		threshold /= blockSize
	}

	return humanize.Commaf(num)
}
