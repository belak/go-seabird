package extra

import (
	"math/big"
	"strings"

	"github.com/soudy/mathcat"

	seabird "github.com/belak/go-seabird"
	irc "gopkg.in/irc.v3"
)

func init() {
	seabird.RegisterPlugin("math", newMathPlugin)
}

func newMathPlugin(cm *seabird.CommandMux) {
	cm.Event("math", exprCallback, &seabird.HelpInfo{
		Usage:       "<expr>",
		Description: "Math. Like calculators and stuff. Bug somebody if you don't know how to math.",
	})
}

func exprCallback(b *seabird.Bot, m *irc.Message) {
	var err error
	var res *big.Rat

	mc := mathcat.New()
	for _, expr := range strings.Split(m.Trailing(), ";") {
		res, err = mc.Run(expr)
		if err != nil {
			b.MentionReply(m, "%s", err)
		}
	}

	b.MentionReply(m, "%s", res.RatString())
}
