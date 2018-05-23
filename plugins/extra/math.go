package extra

import (
	"strings"

	"github.com/soudy/mathcat"

	"github.com/belak/go-seabird"
	irc "github.com/go-irc/irc/v2"
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
