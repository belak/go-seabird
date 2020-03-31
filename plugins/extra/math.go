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

func newMathPlugin(b *seabird.Bot) error {
	cm := b.CommandMux()

	cm.Event("math", exprCallback, &seabird.HelpInfo{
		Usage:       "<expr>",
		Description: "Math. Like calculators and stuff. Bug somebody if you don't know how to math.",
	})

	return nil
}

func exprCallback(r *seabird.Request) {
	var err error

	var res *big.Rat

	var mc = mathcat.New()

	for _, expr := range strings.Split(r.Message.Trailing(), ";") {
		res, err = mc.Run(expr)
		if err != nil {
			r.MentionReplyf("%s", err)
		}
	}

	r.MentionReplyf("%s", res.RatString())
}
