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

type RouletteHandler struct {
	gunSize   int
	shotsLeft int
}

func NewRouletteHandler(gunSize int) *RouletteHandler {
	return &RouletteHandler{gunSize, 0}
}

func (h *RouletteHandler) HandleEvent(c *Client, e *Event) {
	if !e.FromChannel() {
		return
	}

	var msg string

	if h.shotsLeft < 1 {
		h.shotsLeft = rand.Intn(h.gunSize) + 1
		msg = "Reloading the gun... "
	}

	h.shotsLeft -= 1
	if h.shotsLeft < 1 {
		c.ReplyMention(e, "%sBANG!")
		c.Writef("KICK %s %s", e.Args[0], e.Identity.Nick)
	} else {
		c.ReplyMention(e, "Click.")
	}
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
			"KICK %s %s :That's not a valid coin side. Options are: %s",
			e.Args[0],
			e.Identity.Nick,
			strings.Join(coinNames, ", "),
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
