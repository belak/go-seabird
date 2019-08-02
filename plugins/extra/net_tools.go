package extra

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os/exec"

	ping "github.com/belak/go-ping"
	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/plugins/utils"
	"github.com/lrstanley/girc"
)

func init() {
	seabird.RegisterPlugin("nettools", newNetToolsPlugin)
}

type netToolsPlugin struct {
	Key            string
	PrivilegedPing bool
}

func newNetToolsPlugin(b *seabird.Bot, c *girc.Client) error {
	p := &netToolsPlugin{}

	err := b.Config("net_tools", p)
	if err != nil {
		return err
	}

	c.Handlers.AddBg(seabird.PrefixCommand("rdns"), p.RDNS)
	c.Handlers.AddBg(seabird.PrefixCommand("dig"), p.Dig)
	c.Handlers.AddBg(seabird.PrefixCommand("ping"), p.Ping)
	c.Handlers.AddBg(seabird.PrefixCommand("traceroute"), p.Traceroute)
	c.Handlers.AddBg(seabird.PrefixCommand("whois"), p.Whois)
	c.Handlers.AddBg(seabird.PrefixCommand("dnscheck"), p.DNSCheck)
	c.Handlers.AddBg(seabird.PrefixCommand("asn"), p.ASNLookup)

	/*
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
		cm.Event("asn", p.ASNLookup, &seabird.HelpInfo{
			Usage:       "<ip>",
			Description: "Return subnet info for a given IP",
		})
	*/

	return nil
}

func (p *netToolsPlugin) RDNS(c *girc.Client, e girc.Event) {
	go func() {
		if e.Last() == "" {
			c.Cmd.ReplyTof(e, "Argument required")
			return
		}
		names, err := net.LookupAddr(e.Last())
		if err != nil {
			c.Cmd.ReplyTof(e, err.Error())
			return
		}

		if len(names) == 0 {
			c.Cmd.ReplyTof(e, "No results found")
			return
		}

		c.Cmd.ReplyTof(e, names[0])

		if len(names) > 1 {
			for _, name := range names[1:] {
				c.Cmd.Notice(e.Source.Name, name)
			}
		}
	}()
}

func (p *netToolsPlugin) Dig(c *girc.Client, e girc.Event) {
	go func() {
		if e.Last() == "" {
			c.Cmd.ReplyTof(e, "Domain required")
			return
		}

		addrs, err := net.LookupHost(e.Last())
		if err != nil {
			c.Cmd.ReplyTof(e, "%s", err)
			return
		}

		if len(addrs) == 0 {
			c.Cmd.ReplyTof(e, "No results found")
			return
		}

		c.Cmd.ReplyTof(e, addrs[0])

		if len(addrs) > 1 {
			for _, addr := range addrs[1:] {
				c.Cmd.Notice(e.Source.Name, addr)
			}
		}
	}()
}

func (p *netToolsPlugin) Ping(c *girc.Client, e girc.Event) {
	go func() {
		if e.Last() == "" {
			c.Cmd.ReplyTof(e, "Host required")
			return
		}

		pinger, err := ping.NewPinger(e.Last())
		if err != nil {
			c.Cmd.ReplyTof(e, "%s", err)
			return
		}
		pinger.Count = 1
		pinger.SetPrivileged(p.PrivilegedPing)

		pinger.OnRecv = func(pkt *ping.Packet) {
			c.Cmd.ReplyTof(e, "%d bytes from %s: icmp_seq=%d time=%s",
				pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt)
		}
		err = pinger.Run()
		if err != nil {
			c.Cmd.ReplyTof(e, "%s", err)
			return
		}
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

func (p *netToolsPlugin) handleCommand(c *girc.Client, e girc.Event, command string, emptyMsg string) {
	if e.Last() == "" {
		c.Cmd.ReplyTof(e, "Host required")
		return
	}

	url, err := p.runCommand("traceroute", e.Last())
	if err != nil {
		c.Cmd.ReplyTof(e, "%s", err)
		return
	}

	c.Cmd.ReplyTof(e, "%s", url)
}

func (p *netToolsPlugin) Traceroute(c *girc.Client, e girc.Event) {
	go p.handleCommand(c, e, "traceroute", "Host required")
}

func (p *netToolsPlugin) Whois(c *girc.Client, e girc.Event) {
	go p.handleCommand(c, e, "whois", "Domain required")
}

func (p *netToolsPlugin) DNSCheck(c *girc.Client, e girc.Event) {
	if e.Last() == "" {
		c.Cmd.ReplyTof(e, "Domain required")
		return
	}

	c.Cmd.ReplyTof(e, "https://www.whatsmydns.net/#A/"+e.Last())
}

type asnResponse struct {
	Announced     bool
	AsCountryCode string `json:"as_country_code"`
	AsDescription string `json:"as_description"`
	AsNumber      int    `json:"as_number"`
	FirstIP       string `json:"first_ip"`
	LastIP        string `json:"last_ip"`
}

func (p *netToolsPlugin) ASNLookup(c *girc.Client, e girc.Event) {
	if e.Last() == "" {
		c.Cmd.ReplyTof(e, "IP required")
		return
	}

	asnResp := asnResponse{}

	err := utils.GetJSON(
		"https://api.iptoasn.com/v1/as/ip/"+e.Last(),
		&asnResp)
	if err != nil {
		c.Cmd.ReplyTof(e, "%s", err)
		return
	}

	if !asnResp.Announced {
		c.Cmd.ReplyTof(e, "ASN information not available")
		return
	}

	c.Cmd.ReplyTof(e,
		"#%d (%s - %s) - %s (%s)",
		asnResp.AsNumber,
		asnResp.FirstIP,
		asnResp.LastIP,
		asnResp.AsDescription,
		asnResp.AsCountryCode)
}
