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

func newChancePlugin(b *seabird.Bot) error {
	cm := b.CommandMux()

	p := &chancePlugin{
		6,
		make(map[string]int),
	}

	cm.Channel("roulette", p.rouletteCallback, &seabird.HelpInfo{
		Description: "Click... click... BANG!",
	})

	cm.Channel("coin", p.coinCallback, &seabird.HelpInfo{
		Usage:       "[heads|tails]",
		Description: "Guess the coin flip. If you guess wrong, you're out!",
	})

	return nil
}

func (p *chancePlugin) rouletteCallback(r *seabird.Request) {
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
		r.MentionReplyf("%sBANG!", msg)
		r.Writef("KICK %s %s", r.Message.Params[0], r.Message.Prefix.Name)
	} else {
		r.MentionReplyf("%sClick.", msg)
	}

	p.rouletteShotsLeft[r.Message.Params[0]] = shotsLeft
}

func (p *chancePlugin) coinCallback(r *seabird.Request) {
	guess := -1
	guessStr := r.Message.Trailing()

	for k, v := range coinNames {
		if guessStr == v {
			guess = k
			break
		}
	}

	if guess == -1 {
		r.Writef(
			"KICK %s %s :That's not a valid coin side. Options are: %s",
			r.Message.Params[0],
			r.Message.Prefix.Name,
			strings.Join(coinNames, ", "))

		return
	}

	flip := rand.Intn(2)

	if flip == guess {
		r.MentionReplyf("Lucky guess!")
	} else {
		r.Writef("KICK %s %s :%s", r.Message.Params[0], r.Message.Prefix.Name, "Sorry! Better luck next time!")
	}
}
