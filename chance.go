package seabird

import (
	"math/rand"
	"strings"

	"bitbucket.org/belak/irc"
)

var coinNames = []string{
	"heads",
	"tails",
}

func CoinKickHandler(c *irc.Client, e *irc.Event) {
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
		c.Writef(
			"KICK %s %s :%s",
			e.Args[0],
			e.Identity.Nick,
			"That's not a valid coin side. Options are: %s", strings.Join(coinNames, ", "),
		)
		return
	}

	flip := rand.Intn(2)

	if flip == guess {
		c.MentionReply(e, "Lucky guess!")
	} else {
		c.Writef("KICK %s %s :%s", e.Args[0], e.Identity.Nick, "Sorry! Better luck next time!")
	}
}
