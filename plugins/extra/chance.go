package extra

import (
	"math/rand"
	"strings"

	"github.com/lrstanley/girc"

	seabird "github.com/belak/go-seabird"
)

func init() {
	seabird.RegisterPlugin("chance", newChancePlugin)
}

var coinNames = []string{
	"heads",
	"tails",
}

type chancePlugin struct {
	RouletteGunSize   int
	rouletteShotsLeft map[string]int
}

func newChancePlugin(b *seabird.Bot, c *girc.Client) {
	p := &chancePlugin{
		6,
		make(map[string]int),
	}

	c.Handlers.AddBg(seabird.PrefixCommand("roulette"), p.rouletteCallback)
	c.Handlers.AddBg(seabird.PrefixCommand("coin"), p.coinCallback)

	/*
		cm.Event("roulette", p.rouletteCallback, &seabird.HelpInfo{
			Description: "Click... click... BANG!",
		})

		cm.Event("coin", p.coinCallback, &seabird.HelpInfo{
			Usage:       "[heads|tails]",
			Description: "Guess the coin flip. If you guess wrong, you're out!",
		})
	*/
}

func (p *chancePlugin) rouletteCallback(c *girc.Client, e girc.Event) {
	if !e.IsFromChannel() {
		return
	}

	shotsLeft := p.rouletteShotsLeft[e.Params[0]]

	var msg string
	if shotsLeft < 1 {
		shotsLeft = rand.Intn(p.RouletteGunSize) + 1
		msg = "Reloading the gun... "
	}

	shotsLeft--
	if shotsLeft < 1 {
		c.Cmd.ReplyTof(e, "%sBANG!", msg)
		c.Cmd.SendRawf("KICK %s %s", e.Params[0], e.Source.Name)
	} else {
		c.Cmd.ReplyTof(e, "%sClick.", msg)
	}

	p.rouletteShotsLeft[e.Params[0]] = shotsLeft
}

func (p *chancePlugin) coinCallback(c *girc.Client, e girc.Event) {
	c.Cmd.ReplyTo(e, "test")
	if !e.IsFromChannel() {
		return
	}

	guess := -1
	guessStr := e.Last()
	for k, v := range coinNames {
		if guessStr == v {
			guess = k
			break
		}
	}

	if guess == -1 {
		c.Cmd.SendRawf(
			"KICK %s %s :That's not a valid coin side. Options are: %s",
			e.Params[0],
			e.Source.Name,
			strings.Join(coinNames, ", "))
		return
	}

	flip := rand.Intn(2)

	if flip == guess {
		c.Cmd.ReplyTo(e, "Lucky guess!")
	} else {
		c.Cmd.SendRawf("KICK %s %s :%s", e.Params[0], e.Source.Name, "Sorry! Better luck next time!")
	}
}
