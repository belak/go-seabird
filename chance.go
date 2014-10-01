package seabird

import (
	"math/rand"
	"strings"

	"bitbucket.org/belak/seabird/bot"
	"bitbucket.org/belak/seabird/irc"
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

func NewChancePlugin(b *bot.Bot) (bot.Plugin, error) {
	p := &ChancePlugin{}
	err := p.Reload(b)
	if err != nil {
		return nil, err
	}

	b.Command("roulette", "Click... click... BANG!", p.Roulette)
	b.Command("coin", "[heads|tails]", p.Coin)

	return p, nil
}

func (p *ChancePlugin) Reload(b *bot.Bot) error {
	err := b.LoadConfig("chance", p)
	if err != nil {
		return err
	}
	return nil
}

func (p *ChancePlugin) Roulette(b *bot.Bot, e *irc.Event) {
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
		b.MentionReply(e, "%sBANG!", msg)
		b.C.Writef("KICK %s %s", e.Args[0], e.Identity.Nick)
	} else {
		b.MentionReply(e, "%sClick.", msg)
	}
}

func (p *ChancePlugin) Coin(b *bot.Bot, e *irc.Event) {
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
		b.C.Writef(
			"KICK %s %s :That's not a valid coin side. Options are: %s",
			e.Args[0],
			e.Identity.Nick,
			strings.Join(coinNames, ", "),
		)
		return
	}

	flip := rand.Intn(2)

	if flip == guess {
		b.MentionReply(e, "Lucky guess!")
	} else {
		b.C.Writef("KICK %s %s :%s", e.Args[0], e.Identity.Nick, "Sorry! Better luck next time!")
	}
}
