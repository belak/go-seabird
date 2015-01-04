package link_providers

import (
	"github.com/belak/irc"
)

type LinkProvider interface {
	Handle(url string, c *irc.Client, e *irc.Event) bool
}
