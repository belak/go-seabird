package extra

import (
	"strings"

	"github.com/lrstanley/girc"
	"github.com/soudy/mathcat"

	seabird "github.com/belak/go-seabird"
)

func init() {
	seabird.RegisterPlugin("math", newMathPlugin)
}

func newMathPlugin(c *girc.Client) {
	c.Handlers.AddBg(seabird.PrefixCommand("math"), exprCallback)

	/*
		cm.Event("math", exprCallback, &seabird.HelpInfo{
			Usage:       "<expr>",
			Description: "Math. Like calculators and stuff. Bug somebody if you don't know how to math.",
		})
	*/
}

func exprCallback(c *girc.Client, e girc.Event) {
	var err error
	var res float64

	mc := mathcat.New()
	for _, expr := range strings.Split(e.Last(), ";") {
		res, err = mc.Run(expr)
		if err != nil {
			c.Cmd.ReplyTof(e, "%s", err)
		}
	}

	c.Cmd.ReplyTof(e, "%g", res)
}
