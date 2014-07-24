package plugins

import (
	seabird ".."

	"encoding/json"
	"math/rand"
	"strings"

	"github.com/thoj/go-ircevent"
)

var coinNames = []string{
	"heads",
	"tails",
}

type ChancePlugin struct {
	Bot *seabird.Bot
}

func init() {
	seabird.RegisterPlugin("chance", NewChancePlugin)
}

func NewChancePlugin(b *seabird.Bot, c json.RawMessage) {
	p := &ChancePlugin{Bot: b}

	/*
		err := json.Unmarshal(c, &p.Key)
		if err != nil {
			fmt.Println(err)
		}
	*/

	b.RegisterFunction("coin", p.CoinKick)
	//b.RegisterFunction("roulette", p.ForecastCurrent)
}

func (p *ChancePlugin) CoinKick(e *irc.Event) {
	if len(e.Arguments) == 0 || e.Arguments[0][0] != '#' {
		return
	}

	guess := -1
	guessStr := strings.TrimSpace(e.Message())
	for k, v := range coinNames {
		if guessStr == v {
			guess = k
			break
		}
	}

	if guess == -1 {
		p.Bot.MentionReply(e, "That's not a valid coin side")
		return
	}

	flip := rand.Intn(2)

	if flip == guess {
		p.Bot.MentionReply(e, "Lucky guess!")
	} else {
		p.Bot.Conn.SendRawf("KICK %s %s :%s", e.Arguments[0], e.Nick, "Sorry!")
	}
}
