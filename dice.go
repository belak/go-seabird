package seabird

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	"bitbucket.org/belak/irc"
	"bitbucket.org/belak/seabird/bot"
)

func init() {
	bot.RegisterPlugin("dice", NewDicePlugin)
}

var diceRe = regexp.MustCompile(`(?:^|\b)(\d*)d(\d+)\b`)

type DicePlugin struct{}

func NewDicePlugin(b *bot.Bot) (bot.Plugin, error) {
	p := &DicePlugin{}
	b.Mention(p.Dice)
	return p, nil
}

func (p *DicePlugin) Reload(b *bot.Bot) error {
	// noop
	return nil
}

func (p *DicePlugin) Dice(b *bot.Bot, e *irc.Event) {
	var rolls []string
	totalCount := 0

	matches := diceRe.FindAllStringSubmatch(e.Trailing(), -1)
	for _, match := range matches {
		if len(match) != 3 {
			continue
		}

		// Grab the count, otherwise 1
		count, _ := strconv.Atoi(match[1])
		if count == 0 {
			count = 1
		}

		// Clamp count
		if count < 0 {
			b.MentionReply(e, "You cannot request a negative number of rolls")
			return
		}

		totalCount += count
		if totalCount > 100 {
			b.MentionReply(e, "You cannot request more than 100 dice")
			return
		}

		// How big is the die?
		size, _ := strconv.Atoi(match[2])

		if size > 100 {
			b.MentionReply(e, "You cannot request dice larger than 100")
			return
		}

		// Clamp size
		if size < 1 {
			b.MentionReply(e, "You cannot request dice smaller than 1")
			return
		}

		var dice []string
		for i := 0; i < count; i++ {
			dice = append(dice, fmt.Sprintf("%d", rand.Intn(size)+1))
		}

		rolls = append(rolls, fmt.Sprintf("%dd%d: %s", count, size, strings.Join(dice, ", ")))
	}

	if len(rolls) > 0 {
		b.MentionReply(e, "%s", strings.Join(rolls, " "))
	}
}
