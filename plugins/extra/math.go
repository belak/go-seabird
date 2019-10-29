package extra

import (
	"math/big"
	"strings"

	"github.com/soudy/mathcat"

	seabird "github.com/belak/go-seabird"
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

func exprCallback(b *seabird.Bot, r *seabird.Request) {
	var (
		err error
		res *big.Rat

		mc = mathcat.New()
	)

	for _, expr := range strings.Split(r.Message.Trailing(), ";") {
		res, err = mc.Run(expr)
		if err != nil {
			b.MentionReply(r, "%s", err)
		}
	}

	b.MentionReply(r, "%s", res.RatString())
}
