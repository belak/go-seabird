package extra

import (
	"math/rand"
	"strings"

	"github.com/belak/go-seabird"
	"github.com/go-irc/irc"
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

func (p *chancePlugin) rouletteCallback(b *seabird.Bot, m *irc.Message) {
	if !b.FromChannel(m) {
		return
	}

	if len(m.Params) < 1 || len(m.Params[0]) < 1 {
		// Invalid message
		return
	}

	shotsLeft := p.rouletteShotsLeft[m.Params[0]]

	var msg string
	if shotsLeft < 1 {
		shotsLeft = rand.Intn(p.RouletteGunSize) + 1
		msg = "Reloading the gun... "
	}

	shotsLeft--
	if shotsLeft < 1 {
		b.MentionReply(m, "%sBANG!", msg)
		b.Writef("KICK %s %s", m.Params[0], m.Prefix.Name)
	} else {
		b.MentionReply(m, "%sClick.", msg)
	}

	p.rouletteShotsLeft[m.Params[0]] = shotsLeft
}

func (p *chancePlugin) coinCallback(b *seabird.Bot, m *irc.Message) {
	if !b.FromChannel(m) {
		return
	}

	guess := -1
	guessStr := m.Trailing()
	for k, v := range coinNames {
		if guessStr == v {
			guess = k
			break
		}
	}

	if guess == -1 {
		b.Writef(
			"KICK %s %s :That's not a valid coin side. Options are: %s",
			m.Params[0],
			m.Prefix.Name,
			strings.Join(coinNames, ", "),
		)
		return
	}

	flip := rand.Intn(2)

	if flip == guess {
		b.MentionReply(m, "Lucky guess!")
	} else {
		b.Writef("KICK %s %s :%s", m.Params[0], m.Prefix.Name, "Sorry! Better luck next time!")
	}
}
