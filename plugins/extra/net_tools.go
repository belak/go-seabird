package extra

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"

	"github.com/belak/go-seabird"
	"github.com/belak/irc"
)

func init() {
	seabird.RegisterPlugin("nettools", newNetToolsPlugin)
}

type netToolsPlugin struct {
	Key string
}

func newNetToolsPlugin(b *seabird.Bot, cm *seabird.CommandMux) error {
	p := &netToolsPlugin{}

	err := b.Config("net_tools", p)
	if err != nil {
		return err
	}

	cm.Event("rdns", p.RDNS, &seabird.HelpInfo{
		Usage:       "<ip>",
		Description: "Does a reverse DNS lookup on the given IP",
	})
	cm.Event("dig", p.Dig, &seabird.HelpInfo{
		Usage:       "<domain>",
		Description: "Retrieves IP records for given domain",
	})
	cm.Event("ping", p.Ping, &seabird.HelpInfo{
		Usage:       "<host>",
		Description: "Pings given host once",
	})
	cm.Event("traceroute", p.Traceroute, &seabird.HelpInfo{
		Usage:       "<host>",
		Description: "Runs traceroute on given host and returns pastebin URL for results",
	})
	cm.Event("whois", p.Whois, &seabird.HelpInfo{
		Usage:       "<domain>",
		Description: "Runs whois on given domain and returns pastebin URL for results",
	})
	cm.Event("dnscheck", p.DNSCheck, &seabird.HelpInfo{
		Usage:       "<domain>",
		Description: "Returns DNSCheck URL for domain",
	})

	return nil
}

func (p *netToolsPlugin) RDNS(b *seabird.Bot, m *irc.Message) {
	go func() {
		if m.Trailing() == "" {
			b.MentionReply(m, "Argument required")
			return
		}
		names, err := net.LookupAddr(m.Trailing())
		if err != nil {
			b.MentionReply(m, err.Error())
			return
		}

		if len(names) == 0 {
			b.MentionReply(m, "No results found")
			return
		}

		b.MentionReply(m, names[0])

		if len(names) > 1 {
			for _, name := range names[1:] {
				b.Writef("NOTICE %s :%s", m.Prefix.Name, name)
			}
		}
	}()
}

func (p *netToolsPlugin) Dig(b *seabird.Bot, m *irc.Message) {
	go func() {
		if m.Trailing() == "" {
			b.MentionReply(m, "Domain required")
			return
		}

		addrs, err := net.LookupHost(m.Trailing())
		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}

		if len(addrs) == 0 {
			b.MentionReply(m, "No results found")
			return
		}

		b.MentionReply(m, addrs[0])

		if len(addrs) > 1 {
			for _, addr := range addrs[1:] {
				b.Writef("NOTICE %s :%s", m.Prefix.Name, addr)
			}
		}
	}()
}

func (p *netToolsPlugin) Ping(b *seabird.Bot, m *irc.Message) {
	go func() {
		if m.Trailing() == "" {
			b.MentionReply(m, "Host required")
			return
		}

		out, err := exec.Command("ping", "-c1", m.Trailing()).Output()
		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}

		arr := strings.Split(string(out), "\n")
		if len(arr) < 2 {
			b.MentionReply(m, "Error retrieving ping results")
			return
		}

		b.MentionReply(m, arr[1])
	}()
}

func (p *netToolsPlugin) pasteData(data string) (string, error) {
	resp, err := http.PostForm("http://pastebin.com/api/api_post.php", url.Values{
		"api_dev_key":    {p.Key},
		"api_option":     {"paste"},
		"api_paste_code": {data},
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	return string(body), err
}

func (p *netToolsPlugin) runCommand(cmd string, args ...string) (string, error) {
	out, err := exec.Command(cmd, args...).Output()
	if err != nil {
		return "", err
	}

	return p.pasteData(string(out))
}

func (p *netToolsPlugin) handleCommand(b *seabird.Bot, m *irc.Message, command string, emptyMsg string) {
	if m.Trailing() == "" {
		b.MentionReply(m, "Host required")
		return
	}

	url, err := p.runCommand("traceroute", m.Trailing())
	if err != nil {
		b.MentionReply(m, "%s", err)
		return
	}

	b.MentionReply(m, "%s", url)

}

func (p *netToolsPlugin) Traceroute(b *seabird.Bot, m *irc.Message) {
	go p.handleCommand(b, m, "traceroute", "Host required")
}

func (p *netToolsPlugin) Whois(b *seabird.Bot, m *irc.Message) {
	go p.handleCommand(b, m, "whois", "Domain required")
}

func (p *netToolsPlugin) DNSCheck(b *seabird.Bot, m *irc.Message) {
	if m.Trailing() == "" {
		b.MentionReply(m, "Domain required")
		return
	}

	b.MentionReply(m, "https://www.whatsmydns.net/#A/"+m.Trailing())
}
