package link_providers

import (
	"github.com/belak/irc"
)

type LinkProvider interface {
	Handles(url string) bool
	Handle(url string, c *irc.Client, e *irc.Event)
}
