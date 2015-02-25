package plugins

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
)

type NetToolsPlugin struct {
	Key string
}

func init() {
	bot.RegisterPlugin("net_tools", NewNetToolsPlugin)
}

func NewNetToolsPlugin(b *bot.Bot, m *mux.CommandMux) error {
	p := &NetToolsPlugin{}

	b.Config("net_tools", p)

	m.Event("rdns", p.RDNS, &mux.HelpInfo{
		"<ip>",
		"Does a reverse DNS lookup on the given IP",
	})
	m.Event("dig", p.Dig, &mux.HelpInfo{
		"<domain>",
		"Retrieves IP records for given domain",
	})
	m.Event("ping", p.Ping, &mux.HelpInfo{
		"<host>",
		"Pings given host once",
	})
	m.Event("traceroute", p.Traceroute, &mux.HelpInfo{
		"<host>",
		"Runs traceroute on given host and returns pastebin URL for results",
	})
	m.Event("whois", p.Whois, &mux.HelpInfo{
		"<domain>",
		"Runs whois on given domain and returns pastebin URL for results",
	})
	m.Event("dnscheck", p.DnsCheck, &mux.HelpInfo{
		"<domain>",
		"Returns DNSCheck URL for domain",
	})

	return nil
}

func (p *NetToolsPlugin) RDNS(c *irc.Client, e *irc.Event) {
	go func() {
		if e.Trailing() == "" {
			c.MentionReply(e, "Argument required")
			return
		}
		names, err := net.LookupAddr(e.Trailing())
		if err != nil {
			c.MentionReply(e, err.Error())
			return
		}

		if len(names) == 0 {
			c.MentionReply(e, "No results found")
			return
		}

		c.MentionReply(e, names[0])

		if len(names) > 1 {
			for _, name := range names[1:] {
				c.Writef("NOTICE %s :%s", e.Identity.Nick, name)
			}
		}
	}()
}

func (p *NetToolsPlugin) Dig(c *irc.Client, e *irc.Event) {
	go func() {
		if e.Trailing() == "" {
			c.MentionReply(e, "Domain required")
			return
		}

		addrs, err := net.LookupHost(e.Trailing())
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}

		if len(addrs) == 0 {
			c.MentionReply(e, "No results found")
			return
		}

		c.MentionReply(e, addrs[0])

		if len(addrs) > 1 {
			for _, addr := range addrs[1:] {
				c.Writef("NOTICE %s :%s", e.Identity.Nick, addr)
			}
		}
	}()
}

func (p *NetToolsPlugin) Ping(c *irc.Client, e *irc.Event) {
	go func() {
		if e.Trailing() == "" {
			c.MentionReply(e, "Host required")
			return
		}

		out, err := exec.Command("ping", "-c1", e.Trailing()).Output()
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}

		arr := strings.Split(string(out), "\n")
		if len(arr) < 2 {
			c.MentionReply(e, "Error retrieving ping results")
			return
		}

		c.MentionReply(e, arr[1])
	}()
}

func (p *NetToolsPlugin) Traceroute(c *irc.Client, e *irc.Event) {
	go func() {
		if e.Trailing() == "" {
			c.MentionReply(e, "Host required")
			return
		}

		out, err := exec.Command("traceroute", e.Trailing()).Output()
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}

		resp, err := http.PostForm("http://pastebin.com/api/api_post.php", url.Values{
			"api_dev_key":    {p.Key},
			"api_option":     {"paste"},
			"api_paste_code": {string(out)},
		})
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}

		c.MentionReply(e, "%s", body)
	}()
}

func (p *NetToolsPlugin) Whois(c *irc.Client, e *irc.Event) {
	go func() {
		if e.Trailing() == "" {
			c.MentionReply(e, "Domain required")
			return
		}

		out, err := exec.Command("whois", e.Trailing()).Output()
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}

		resp, err := http.PostForm("http://pastebin.com/api/api_post.php", url.Values{
			"api_dev_key":    {p.Key},
			"api_option":     {"paste"},
			"api_paste_code": {string(out)},
		})
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}

		c.MentionReply(e, "%s", body)
	}()
}

func (p *NetToolsPlugin) DnsCheck(c *irc.Client, e *irc.Event) {
	// Just for Kaleb
	go func() {
		if e.Trailing() == "" {
			c.MentionReply(e, "Domain required")
			return
		}

		c.MentionReply(e, "https://www.whatsmydns.net/#A/"+e.Trailing())
	}()
}
