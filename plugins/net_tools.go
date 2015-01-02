package plugins

import (
	"net"
	"os/exec"
	"strings"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
)

func init() {
	bot.RegisterPlugin("net_tools", NewNetToolsPlugin)
}

func NewNetToolsPlugin(m *mux.CommandMux) error {
	m.Event("dig", Dig)
	m.Event("ping", Ping)
	m.Event("dnscheck", DnsCheck)

	return nil
}

func Dig(c *irc.Client, e *irc.Event) {
	if e.Trailing() == "" {
		c.MentionReply(e, "Domain required")
		return
	}

	addrs, err := net.LookupHost(e.Trailing())
	if err != nil {
		c.MentionReply(e, err.Error())
	}

	c.MentionReply(e, addrs[0])
}

func Ping(c *irc.Client, e *irc.Event) {
	if e.Trailing() == "" {
		c.MentionReply(e, "Host required")
		return
	}

	out, err := exec.Command("ping", "-c1", e.Trailing()).Output()
	if err != nil {
		c.MentionReply(e, err.Error())
		return
	}

	result := strings.Split(string(out), "\n")[1]
	c.MentionReply(e, result)
}

func DnsCheck(c *irc.Client, e *irc.Event) {
	if e.Trailing() == "" {
		c.MentionReply(e, "Domain required")
		return
	}

	c.MentionReply(e, "https://www.whatsmydns.net/#A/" + e.Trailing())
}
