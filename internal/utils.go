package internal

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/sirupsen/logrus"
)

// TemplateMustCompile will add all the helpers to a new template,
// compile it and panic if that fails. Note that it will also trim
// space from the start and end of the template to make definitions
// easier.
//
// Provided functions:
// - dateFormat - takes one argument, the format of the date (in golang format)
// - pluralize - takes one argument, the number of something this is describing
func TemplateMustCompile(name, data string) *template.Template {
	ret := template.New(name)
	ret.Funcs(template.FuncMap{
		"dateFormat":     dateFormat,
		"pluralize":      templatePluralize,
		"pluralizeWord":  templatePluralizeWord,
		"prettifySuffix": templatePrettifySuffix,
	})

	template.Must(ret.Parse(strings.TrimSpace(data)))

	return ret
}

// RenderTemplate is a wrapper to render a template to a string.
func RenderTemplate(t *template.Template, vars interface{}) (string, error) {
	b := bytes.NewBuffer(nil)

	err := t.Execute(b, vars)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

// RenderRespond is a wrapper around RenderTemplate which will render a template
// and respond to the given message. It will return true on success and false on
// failure.
func RenderRespond(r func(format string, v ...interface{}) error, logger *logrus.Entry, t *template.Template, prefix string, vars interface{}) bool {
	out, err := RenderTemplate(t, vars)
	if err != nil {
		logger.WithError(err).Error("Failed to render template")
		return false
	}

	r("%s %s", prefix, out)

	return true
}

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
