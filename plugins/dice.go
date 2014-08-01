package plugins

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	irc "github.com/thoj/go-ircevent"

	seabird ".."
)

func init() {
	seabird.RegisterPlugin("dice", NewDicePlugin)
}

type DicePlugin struct {
	Bot *seabird.Bot
}

var diceRe = regexp.MustCompile(`(?:^|\b)(\d*)d(\d+)\b`)

func NewDicePlugin(b *seabird.Bot, c json.RawMessage) {
	p := &DicePlugin{b}
	b.RegisterCallback("PRIVMSG", p.Msg)
}

func (p *DicePlugin) Msg(e *irc.Event) {
	var rolls []string
	totalCount := 0

	matches := diceRe.FindAllStringSubmatch(e.Message(), -1)
	for _, match := range matches {
		if len(match) != 3 {
			continue
		}

		// Grab the count, otherwise 1
		count, _ := strconv.Atoi(match[1])

		// Clamp count
		if count < 1 {
			p.Bot.MentionReply(e, "You cannot request a non-positive number of rolls")
			return
		}

		totalCount += count
		if totalCount > 100 {
			p.Bot.MentionReply(e, "You cannot request more than 100 dice")
			return
		}

		// How big is the die?
		size, _ := strconv.Atoi(match[2])

		if size > 100 {
			p.Bot.MentionReply(e, "You cannot request dice larger than 100")
			return
		}

		// Clamp size
		if size < 1 {
			p.Bot.MentionReply(e, "You cannot request a non-positive die size")
			return
		}

		var dice []string
		for i := 0; i < count; i++ {
			dice = append(dice, fmt.Sprintf("%d", rand.Intn(size)+1))
		}

		rolls = append(rolls, fmt.Sprintf("%dd%d: %s", count, size, strings.Join(dice, ", ")))
	}

	if len(rolls) > 0 {
		p.Bot.MentionReply(e, "%s", strings.Join(rolls, " "))
	}
}
