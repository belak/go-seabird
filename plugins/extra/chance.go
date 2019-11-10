package extra

import (
	"math/rand"
	"strings"

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

func newChancePlugin(b *seabird.Bot, cm *seabird.CommandMux) {
	p := &chancePlugin{
		6,
		make(map[string]int),
	}

	cm.Event("roulette", p.rouletteCallback, &seabird.HelpInfo{
		Description: "Click... click... BANG!",
	})

	cm.Event("coin", p.coinCallback, &seabird.HelpInfo{
		Usage:       "[heads|tails]",
		Description: "Guess the coin flip. If you guess wrong, you're out!",
	})
}

func (p *chancePlugin) rouletteCallback(b *seabird.Bot, r *seabird.Request) {
	if !r.FromChannel() {
		return
	}

	if len(r.Message.Params) < 1 || len(r.Message.Params[0]) < 1 {
		// Invalid message
		return
	}

	shotsLeft := p.rouletteShotsLeft[r.Message.Params[0]]

	var msg string

	if shotsLeft < 1 {
		shotsLeft = rand.Intn(p.RouletteGunSize) + 1
		msg = "Reloading the gun... "
	}

	shotsLeft--

	if shotsLeft < 1 {
		r.MentionReply("%sBANG!", msg)
		b.Writef("KICK %s %s", r.Message.Params[0], r.Message.Prefix.Name)
	} else {
		r.MentionReply("%sClick.", msg)
	}

	p.rouletteShotsLeft[r.Message.Params[0]] = shotsLeft
}

func (p *chancePlugin) coinCallback(b *seabird.Bot, r *seabird.Request) {
	if !r.FromChannel() {
		return
	}

	guess := -1
	guessStr := r.Message.Trailing()

	for k, v := range coinNames {
		if guessStr == v {
			guess = k
			break
		}
	}

	if guess == -1 {
		b.Writef(
			"KICK %s %s :That's not a valid coin side. Options are: %s",
			r.Message.Params[0],
			r.Message.Prefix.Name,
			strings.Join(coinNames, ", "))

		return
	}

	flip := rand.Intn(2)

	if flip == guess {
		r.MentionReply("Lucky guess!")
	} else {
		b.Writef("KICK %s %s :%s", r.Message.Params[0], r.Message.Prefix.Name, "Sorry! Better luck next time!")
	}
}
