package nettools

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os/exec"

	ping "github.com/belak/go-ping"
	"github.com/belak/go-seabird/internal"

	seabird "github.com/belak/go-seabird"
)

func init() {
	seabird.RegisterPlugin("nettools", newNetToolsPlugin)
}

type netToolsPlugin struct {
	Key            string
	PrivilegedPing bool
}

func newNetToolsPlugin(b *seabird.Bot) error {
	p := &netToolsPlugin{}

	err := b.Config("net_tools", p)
	if err != nil {
		return err
	}

	cm := b.CommandMux()

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

	return nil
}

func (p *netToolsPlugin) RDNS(r *seabird.Request) {
	go func() {
		if r.Message.Trailing() == "" {
			r.MentionReplyf("Argument required")
			return
		}
		names, err := net.LookupAddr(r.Message.Trailing())
		if err != nil {
			r.MentionReplyf(err.Error())
			return
		}

		if len(names) == 0 {
			r.MentionReplyf("No results found")
			return
		}

		r.MentionReplyf(names[0])

		if len(names) > 1 {
			for _, name := range names[1:] {
				r.Writef("NOTICE %s :%s", r.Message.Prefix.Name, name)
			}
		}
	}()
}

func (p *netToolsPlugin) Dig(r *seabird.Request) {
	go func() {
		if r.Message.Trailing() == "" {
			r.MentionReplyf("Domain required")
			return
		}

		addrs, err := net.LookupHost(r.Message.Trailing())
		if err != nil {
			r.MentionReplyf("%s", err)
			return
		}

		if len(addrs) == 0 {
			r.MentionReplyf("No results found")
			return
		}

		r.MentionReplyf(addrs[0])

		if len(addrs) > 1 {
			for _, addr := range addrs[1:] {
				r.Writef("NOTICE %s :%s", r.Message.Prefix.Name, addr)
			}
		}
	}()
}

func (p *netToolsPlugin) Ping(r *seabird.Request) {
	go func() {
		if r.Message.Trailing() == "" {
			r.MentionReplyf("Host required")
			return
		}

		pinger, err := ping.NewPinger(r.Message.Trailing())
		if err != nil {
			r.MentionReplyf("%s", err)
			return
		}
		pinger.Count = 1
		pinger.SetPrivileged(p.PrivilegedPing)

		pinger.OnRecv = func(pkt *ping.Packet) {
			r.MentionReplyf("%d bytes from %s: icmp_seq=%d time=%s",
				pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt)
		}
		err = pinger.Run()
		if err != nil {
			r.MentionReplyf("%s", err)
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

func (p *netToolsPlugin) handleCommand(r *seabird.Request, command string, emptyMsg string) {
	if r.Message.Trailing() == "" {
		r.MentionReplyf("Host required")
		return
	}

	url, err := p.runCommand(command, r.Message.Trailing())
	if err != nil {
		r.MentionReplyf("%s", err)
		return
	}

	r.MentionReplyf("%s", url)
}

func (p *netToolsPlugin) Traceroute(r *seabird.Request) {
	go p.handleCommand(r, "traceroute", "Host required")
}

func (p *netToolsPlugin) Whois(r *seabird.Request) {
	go p.handleCommand(r, "whois", "Domain required")
}

func (p *netToolsPlugin) DNSCheck(r *seabird.Request) {
	if r.Message.Trailing() == "" {
		r.MentionReplyf("Domain required")
		return
	}

	r.MentionReplyf("https://www.whatsmydns.net/#A/" + r.Message.Trailing())
}

type asnResponse struct {
	Announced     bool
	AsCountryCode string `json:"as_country_code"`
	AsDescription string `json:"as_description"`
	AsNumber      int    `json:"as_number"`
	FirstIP       string `json:"first_ip"`
	LastIP        string `json:"last_ip"`
}

func (p *netToolsPlugin) ASNLookup(r *seabird.Request) {
	if r.Message.Trailing() == "" {
		r.MentionReplyf("IP required")
		return
	}

	asnResp := asnResponse{}

	err := internal.GetJSON(
		"https://api.iptoasn.com/v1/as/ip/"+r.Message.Trailing(),
		&asnResp)
	if err != nil {
		r.MentionReplyf("%s", err)
		return
	}

	if !asnResp.Announced {
		r.MentionReplyf("ASN information not available")
		return
	}

	r.MentionReplyf(
		"#%d (%s - %s) - %s (%s)",
		asnResp.AsNumber,
		asnResp.FirstIP,
		asnResp.LastIP,
		asnResp.AsDescription,
		asnResp.AsCountryCode)
}
