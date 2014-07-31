package plugins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"

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
	var output bytes.Buffer

	first := true
	matches := diceRe.FindAllStringSubmatch(e.Message(), -1)
	for _, match := range matches {
		if len(match) != 3 {
			continue
		}

		// Grab the count, otherwise 1
		count, err := strconv.Atoi(match[1])
		if err != nil {
			count = 1
		}

		// How big is the die?
		size, _ := strconv.Atoi(match[2])

		if !first {
			output.WriteString(" ")
		}

		output.WriteString(fmt.Sprintf("%dd%d: %d", count, size, rand.Intn(size)+1))
		for i := 1; i < count; i++ {
			output.WriteString(fmt.Sprintf(", %d", rand.Intn(size)+1))
		}

		if first {
			first = !first
		}
	}

	if output.Len() > 0 {
		p.Bot.Reply(e, output.String())
	}
}
