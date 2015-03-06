package plugins

//go:generate go tool yacc -o expr_y.go expr.y

import (
	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
)

type MathPlugin struct{}

func NewMathPlugin() bot.Plugin {
	return &MathPlugin{}
}

func (p *MathPlugin) Register(b *bot.Bot) error {
	b.CommandMux.Event("math", p.Expr, &mux.HelpInfo{
		"<expr>",
		"Math. Like calculators and stuff. Bug somebody if you don't know how to math.",
	})
	return nil
}

func (p *MathPlugin) Expr(c *irc.Client, e *irc.Event) {
	val, err := parseExpr(e.Trailing())
	if err != nil {
		c.Reply(e, "%s", err.Error())
		return
	}

	c.Reply(e, "%s=%g", e.Trailing(), val)
}
