package plugins

import (
	"strings"

	"github.com/soudy/mathcat"

	"github.com/belak/go-seabird/bot"
	"github.com/belak/irc"
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
	var err error
	var res float64

	mc := mathcat.New()
	for _, expr := range strings.Split(m.Trailing(), ";") {
		res, err = mc.Run(expr)
		if err != nil {
			b.MentionReply(m, "%s", err)
		}
	}

	b.MentionReply(m, "%g", res)
}
