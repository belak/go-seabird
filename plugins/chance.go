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
	rouletteShotsLeft map[string]int
}

func NewChancePlugin(b *bot.Bot, m *mux.CommandMux) error {
	p := &ChancePlugin{
		6,
		make(map[string]int),
	}

	m.Event("roulette", p.Roulette) // "Click... click... BANG!"
	m.Event("coin", p.Coin)         // "[heads|tails]"

	return nil
}

func (p *ChancePlugin) Roulette(c *irc.Client, e *irc.Event) {
	if !e.FromChannel() {
		return
	}

	if len(e.Args) < 1 || len(e.Args[0]) < 1 {
		// Invalid message
		return
	}

	shotsLeft := p.rouletteShotsLeft[e.Args[0]]

	var msg string
	if shotsLeft < 1 {
		shotsLeft = rand.Intn(p.RouletteGunSize) + 1
		msg = "Reloading the gun... "
	}

	shotsLeft -= 1
	if shotsLeft < 1 {
		c.MentionReply(e, "%sBANG!", msg)
		c.Writef("KICK %s %s", e.Args[0], e.Identity.Nick)
	} else {
		c.MentionReply(e, "%sClick.", msg)
	}

	p.rouletteShotsLeft[e.Args[0]] = shotsLeft
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
