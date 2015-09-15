package plugins

//go:generate go tool yacc -o expr_y.go expr.y

import (
	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
)

func init() {
	bot.RegisterPlugin("math", NewMathPlugin)
}

type MathPlugin struct{}

func NewMathPlugin(b *bot.Bot) (bot.Plugin, error) {
	p := &MathPlugin{}

	b.CommandMux.Event("math", p.Expr, &bot.HelpInfo{
		Usage:       "<expr>",
		Description: "Math. Like calculators and stuff. Bug somebody if you don't know how to math.",
	})

	return p, nil
}

func (p *MathPlugin) Expr(b *bot.Bot, m *irc.Message) {
	val, err := parseExpr(m.Trailing())
	if err != nil {
		b.Reply(m, "%s", err.Error())
		return
	}

	b.Reply(m, "%s=%g", m.Trailing(), val)
}
