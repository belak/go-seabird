package linkproviders

import (
	"bytes"
	"strings"
	"text/template"
	"time"

	"github.com/bep/inflect"
	"github.com/google/go-github/github"
	"github.com/spf13/cast"
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
		"dateFormat": dateFormat,
		"pluralize":  pluralize,
	})

	template.Must(ret.Parse(strings.TrimSpace(data)))

	return ret
}

// RenderTemplate is a wrapper to render a template to a string.
func RenderTemplate(t *template.Template, vars map[string]interface{}) (string, error) {
	b := bytes.NewBuffer(nil)

	err := t.Execute(b, vars)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

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

func pluralize(count int, in interface{}) (string, error) {
	word, err := cast.ToStringE(in)
	if err != nil {
		return "", err
	}

	if count == 1 {
		return word, nil
	}

	return inflect.Pluralize(word), nil
}
