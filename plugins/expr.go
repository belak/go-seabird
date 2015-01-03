package plugins

//go:generate go tool yacc -o expr_y.go expr.y

import (
	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
)

func init() {
	bot.RegisterPlugin("math", NewMathPlugin)
}

func NewMathPlugin(m *mux.CommandMux) error {
	m.Event("math", mathExpr, &mux.HelpInfo{
		"<expr>",
		"Math. Like calculators and stuff. Bug somebody if you don't know how to math.",
	})
	return nil
}

func mathExpr(c *irc.Client, e *irc.Event) {
	val, err := parseExpr(e.Trailing())
	if err != nil {
		c.Reply(e, "%s", err.Error())
		return
	}

	c.Reply(e, "%s=%g", e.Trailing(), val)
}
