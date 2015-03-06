package plugins

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
)

var diceRe = regexp.MustCompile(`(?:^|\b)(\d*)d(\d+)\b`)

type DicePlugin struct{}

func NewDicePlugin() bot.Plugin {
	return &DicePlugin{}
}

func (p *DicePlugin) Register(b *bot.Bot) error {
	b.MentionMux.Event(p.Dice)
	return nil
}

func (p *DicePlugin) Dice(c *irc.Client, e *irc.Event) {
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
			c.MentionReply(e, "You cannot request a negative number of rolls")
			return
		}

		totalCount += count
		if totalCount > 100 {
			c.MentionReply(e, "You cannot request more than 100 dice")
			return
		}

		// How big is the die?
		size, _ := strconv.Atoi(match[2])

		if size > 100 {
			c.MentionReply(e, "You cannot request dice larger than 100")
			return
		}

		// Clamp size
		if size < 1 {
			c.MentionReply(e, "You cannot request dice smaller than 1")
			return
		}

		var dice []string
		for i := 0; i < count; i++ {
			dice = append(dice, fmt.Sprintf("%d", rand.Intn(size)+1))
		}

		rolls = append(rolls, fmt.Sprintf("%dd%d: %s", count, size, strings.Join(dice, ", ")))
	}

	if len(rolls) > 0 {
		c.MentionReply(e, "%s", strings.Join(rolls, " "))
	}
}
