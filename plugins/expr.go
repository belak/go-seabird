package plugins

//go:generate go tool yacc -o expr_y.go expr.y

import (
	"github.com/belak/seabird/bot"
	"github.com/belak/sorcix-irc"
)

type MathPlugin struct{}

func NewMathPlugin() bot.Plugin {
	return &MathPlugin{}
}

func (p *MathPlugin) Register(b *bot.Bot) error {
	b.CommandMux.Event("math", p.Expr, &bot.HelpInfo{
		"<expr>",
		"Math. Like calculators and stuff. Bug somebody if you don't know how to math.",
	})
	return nil
}

func (p *MathPlugin) Expr(b *bot.Bot, m *irc.Message) {
	val, err := parseExpr(m.Trailing())
	if err != nil {
		b.Reply(m, "%s", err.Error())
		return
	}

	b.Reply(m, "%s=%g", m.Trailing(), val)
}
