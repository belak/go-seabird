package plugins

import (
	"regexp"
	"strings"

	"github.com/belak/seabird/bot"
	"github.com/belak/sorcix-irc"
)

func init() {
	bot.RegisterPlugin("nicktracker", NewNickTrackerPlugin)
}

type NickTrackerPlugin struct {
	modes map[string]string
}

type ModeState int

const (
	ModeNone ModeState = iota
	ModePlus
	ModeMinus
)

func NewNickTrackerPlugin(b *bot.Bot) (bot.Plugin, error) {
	p := &NickTrackerPlugin{
		make(map[string]string),
	}

	b.BasicMux.Event("001", p.Welcome)
	b.BasicMux.Event("005", p.Support)
	b.BasicMux.Event("352", p.Who)
	b.BasicMux.Event("JOIN", p.Join)
	b.BasicMux.Event("PART", p.Part)
	b.BasicMux.Event("MODE", p.Mode)
	b.BasicMux.Event("NICK", p.Nick)

	return p, nil
}

func (p *NickTrackerPlugin) Welcome(b *bot.Bot, m *irc.Message) {
	b.DB.Exec("DELETE FROM nicks")

	// Request all prefixes to be reported
	b.Writef("CAP REQ multi-prefix")
}

func (p *NickTrackerPlugin) Support(b *bot.Bot, m *irc.Message) {
	// Format: PREFIX=(ov)@+
	prefix := regexp.MustCompile(`^PREFIX=\(([A-Za-z]+)\)([@&%~+]+)$`)

	for _, arg := range m.Params {
		matches := prefix.FindStringSubmatch(arg)
		if len(matches) == 3 {
			p.modes = make(map[string]string)
			modes := matches[1]
			flags := matches[2]
			for i, char := range modes {
				p.modes[string(flags[i])] = string(char)
			}
		}
	}
}

func (p *NickTrackerPlugin) Who(b *bot.Bot, m *irc.Message) {
	// Format: sinisalo.freenode.net 352 starkbot #encoded ~belak encoded/developer/belak kornbluth.freenode.net belak G+ :0 Kaleb Elwert
	if len(m.Params) < 7 {
		// Modes are index 6
		return
	}

	// Mode format: H@+
	channel := m.Params[1]
	nick := m.Params[5]
	flags := m.Params[6][1:]
	modes := ""
	for _, ch := range flags {
		modes += p.modes[string(ch)]
	}

	p.addNick(b, channel, nick, modes)
}

func (p *NickTrackerPlugin) Join(b *bot.Bot, m *irc.Message) {
	if len(m.Params) == 0 {
		// No channel
		return
	}

	// Check if it's a JOIN for the bot
	if m.Prefix.Name == b.CurrentNick() {
		// Fire off a WHO
		b.Writef("WHO %s", m.Params[0])
	} else {
		p.addNick(b, m.Params[0], m.Prefix.Name, "")
	}
}

func (p *NickTrackerPlugin) Part(b *bot.Bot, m *irc.Message) {
	// Delete nick+channel entry from table
	if len(m.Params) == 0 {
		// No channel
		return
	}

	b.DB.Exec("DELETE FROM nicks WHERE nick=$1 AND channel=$2", m.Prefix.Name, m.Params[0])
}

func (p *NickTrackerPlugin) Mode(b *bot.Bot, m *irc.Message) {
	// Format: :ChanServ!ChanServ@services. MODE #encoded +v starkbot
	if len(m.Params) < 3 {
		return
	}

	channel := m.Params[0]
	nick := m.Params[2]
	changedModes := m.Params[1]

	var modes string
	err := b.DB.Get(&modes, "SELECT flags FROM nicks WHERE channel=$1 AND nick=$2", channel, nick)
	if err != nil {
		return
	}

	// Parse new modes
	state := ModeNone
	for _, ch := range changedModes {
		if state == ModeNone {
			if ch == '+' {
				state = ModePlus
			} else if ch == '-' {
				state = ModeMinus
			} else {
				// String needs to start with + or -
				return
			}
		} else if state == ModePlus {
			if ch == '+' {
				continue
			} else if ch == '-' {
				state = ModeMinus
			} else {
				if !strings.ContainsRune(modes, ch) {
					modes += string(ch)
				}
			}
		} else if state == ModeMinus {
			if ch == '+' {
				state = ModePlus
			} else if ch == '-' {
				continue
			} else {
				modes = strings.Replace(modes, string(ch), "", -1)
			}
		}
	}

	b.DB.Exec("UPDATE nicks SET flags=$1 WHERE channel=$2 AND nick=$3", modes, channel, nick)
}

func (p *NickTrackerPlugin) Nick(b *bot.Bot, m *irc.Message) {
	// Format: :jsvana!~jsvana@encoded/developer/jsvana NICK :foobarbaz
	if len(m.Params) < 1 {
		// No nick
		return
	}

	if m.Prefix.Name == "" {
		// No nick
		return
	}

	oldNick := m.Prefix.Name
	newNick := m.Params[0]

	b.DB.Exec("UPDATE nicks SET nick=$1 WHERE nick=$2", newNick, oldNick)
}

func (p *NickTrackerPlugin) addNick(b *bot.Bot, channel, nick, modes string) {
	//nickRegex := regexp.MustCompile(`(?i)([@~+%&]?)([a-z_\-\[\]\\^{}|`+"`"+`][a-z0-9_\-\[\]\\^{}|`+"`"+`]*)$`)

	b.DB.Exec("INSERT INTO nicks VALUES ($1, $2, $3)", nick, channel, modes)
}
