package plugins

//go:generate go tool yacc -o expr_y.go expr.y

import (
	"github.com/belak/irc"
	"github.com/belak/go-seabird/bot"
)

func init() {
	bot.RegisterPlugin("math", NewMathPlugin)
}

func NewMathPlugin(b *bot.Bot) (bot.Plugin, error) {
	b.CommandMux.Event("math", exprCallback, &bot.HelpInfo{
		Usage:       "<expr>",
		Description: "Math. Like calculators and stuff. Bug somebody if you don't know how to math.",
	})

	return nil, nil
}

func exprCallback(b *bot.Bot, m *irc.Message) {
	val, err := parseExpr(m.Trailing())
	if err != nil {
		b.Reply(m, "%s", err.Error())
		return
	}

	b.Reply(m, "%s=%g", m.Trailing(), val)
}
