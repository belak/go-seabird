package plugins

import (
	"math/rand"
	"strings"

	"github.com/belak/seabird/bot"
	"github.com/belak/irc"
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

func NewChancePlugin(b *bot.Bot) (bot.Plugin, error) {
	p := &ChancePlugin{
		6,
		make(map[string]int),
	}

	b.CommandMux.Event("roulette", p.Roulette, &bot.HelpInfo{
		Description: "Click... click... BANG!",
	})
	b.CommandMux.Event("coin", p.Coin, &bot.HelpInfo{
		"[heads|tails]",
		"Guess the coin flip. If you guess wrong, you're out!",
	})

	return p, nil
}

func (p *ChancePlugin) Roulette(b *bot.Bot, m *irc.Message) {
	if !m.FromChannel() {
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

	shotsLeft -= 1
	if shotsLeft < 1 {
		b.MentionReply(m, "%sBANG!", msg)
		b.Writef("KICK %s %s", m.Params[0], m.Prefix.Name)
	} else {
		b.MentionReply(m, "%sClick.", msg)
	}

	p.rouletteShotsLeft[m.Params[0]] = shotsLeft
}

func (p *ChancePlugin) Coin(b *bot.Bot, m *irc.Message) {
	if !m.FromChannel() {
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
