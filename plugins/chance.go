package plugins

import (
	"math/rand"
	"strings"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
)

func init() {
	bot.RegisterPlugin("chance", NewChancePlugin)
}

var coinNames = []string{
	"heads",
	"tails",
}

type ChancePlugin struct {
	RouletteGunSize   int
	rouletteShotsLeft int
}

func NewChancePlugin(b *bot.Bot, m *mux.CommandMux) (bot.Plugin, error) {
	p := &ChancePlugin{
		6,
		0,
	}

	m.Event("roulette", p.Roulette) // "Click... click... BANG!"
	m.Event("coin", p.Coin)         // "[heads|tails]"

	return p, nil
}

func (p *ChancePlugin) Roulette(c *irc.Client, e *irc.Event) {
	if !e.FromChannel() {
		return
	}

	var msg string
	if p.rouletteShotsLeft < 1 {
		p.rouletteShotsLeft = rand.Intn(p.RouletteGunSize) + 1
		msg = "Reloading the gun... "
	}

	p.rouletteShotsLeft -= 1
	if p.rouletteShotsLeft < 1 {
		c.MentionReply(e, "%sBANG!", msg)
		c.Writef("KICK %s %s", e.Args[0], e.Identity.Nick)
	} else {
		c.MentionReply(e, "%sClick.", msg)
	}
}

func (p *ChancePlugin) Coin(c *irc.Client, e *irc.Event) {
	if !e.FromChannel() {
		return
	}

	guess := -1
	guessStr := strings.TrimSpace(e.Trailing())
	for k, v := range coinNames {
		if guessStr == v {
			guess = k
			break
		}
	}

	if guess == -1 {
		c.Writef(
			"KICK %s %s :That's not a valid coin side. Options are: %s",
			e.Args[0],
			e.Identity.Nick,
			strings.Join(coinNames, ", "),
		)
		return
	}

	flip := rand.Intn(2)

	if flip == guess {
		c.MentionReply(e, "Lucky guess!")
	} else {
		c.Writef("KICK %s %s :%s", e.Args[0], e.Identity.Nick, "Sorry! Better luck next time!")
	}
}
