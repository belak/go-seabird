package plugins

import (
	"regexp"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
)

type NickTrackerPlugin struct {
	db    *sqlx.DB
	modes map[string]string
}

type ModeState int

const (
	ModeNone ModeState = iota
	ModePlus
	ModeMinus
)

func init() {
	bot.RegisterPlugin("nicktracker", NewNickTrackerPlugin)
}

func NewNickTrackerPlugin(b *bot.Bot, bm *irc.BasicMux, m *mux.CommandMux, db *sqlx.DB) (*NickTrackerPlugin, error) {
	p := &NickTrackerPlugin{
		db,
		make(map[string]string),
	}

	bm.Event("001", p.Welcome)
	bm.Event("005", p.Support)
	bm.Event("352", p.Who)
	bm.Event("JOIN", p.Join)
	bm.Event("PART", p.Part)
	bm.Event("MODE", p.Mode)
	bm.Event("NICK", p.Nick)

	return p, nil
}

func (p *NickTrackerPlugin) Welcome(c *irc.Client, e *irc.Event) {
	p.db.Exec("DELETE FROM nicks")

	// Request all prefixes to be reported
	c.Writef("CAP REQ multi-prefix")
}

func (p *NickTrackerPlugin) Support(c *irc.Client, e *irc.Event) {
	// Format: PREFIX=(ov)@+
	prefix := regexp.MustCompile(`^PREFIX=\(([A-Za-z]+)\)([@&%~+]+)$`)

	for _, arg := range e.Args {
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

func (p *NickTrackerPlugin) Who(c *irc.Client, e *irc.Event) {
	// Format: sinisalo.freenode.net 352 starkbot #encoded ~belak encoded/developer/belak kornbluth.freenode.net belak G+ :0 Kaleb Elwert
	if len(e.Args) < 7 {
		// Modes are index 6
		return
	}

	// Mode format: H@+
	channel := e.Args[1]
	nick := e.Args[5]
	flags := e.Args[6][1:]
	modes := ""
	for _, ch := range flags {
		modes += p.modes[string(ch)]
	}

	p.addNick(channel, nick, modes)
}

func (p *NickTrackerPlugin) Join(c *irc.Client, e *irc.Event) {
	if len(e.Args) == 0 {
		// No channel
		return
	}

	// Check if it's a JOIN for the bot
	if e.Identity.Nick == c.CurrentNick() {
		// Fire off a WHO
		c.Writef("WHO %s", e.Args[0])
	} else {
		p.addNick(e.Args[0], e.Identity.Nick, "")
	}
}

func (p *NickTrackerPlugin) Part(c *irc.Client, e *irc.Event) {
	// Delete nick+channel entry from table
	if len(e.Args) == 0 {
		// No channel
		return
	}

	p.db.Exec("DELETE FROM nicks WHERE nick=$1 AND channel=$2", e.Identity.Nick, e.Args[0])
}

func (p *NickTrackerPlugin) Mode(c *irc.Client, e *irc.Event) {
	// Format: :ChanServ!ChanServ@services. MODE #encoded +v starkbot
	if len(e.Args) < 3 {
		return
	}

	channel := e.Args[0]
	nick := e.Args[2]
	changedModes := e.Args[1]

	var modes string
	err := p.db.Get(&modes, "SELECT flags FROM nicks WHERE channel=$1 AND nick=$2", channel, nick)
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

	p.db.Exec("UPDATE nicks SET flags=$1 WHERE channel=$2 AND nick=$3", modes, channel, nick)
}

func (p *NickTrackerPlugin) Nick(c *irc.Client, e *irc.Event) {
	// Format: :jsvana!~jsvana@encoded/developer/jsvana NICK :foobarbaz
	if len(e.Args) < 1 {
		// No nick
		return
	}

	nickRegex := regexp.MustCompile(`(?i)^([a-z_\-\[\]\\^{}|` + "`" + `][a-z0-9_\-\[\]\\^{}|` + "`" + `]*)!`)
	matches := nickRegex.FindStringSubmatch(e.Prefix)
	if len(matches) < 2 {
		// No nick
		return
	}

	oldNick := matches[1]
	newNick := e.Args[0]

	p.db.Exec("UPDATE nicks SET nick=$1 WHERE nick=$2", newNick, oldNick)
}

func (p *NickTrackerPlugin) addNick(channel, nick, modes string) {
	//nickRegex := regexp.MustCompile(`(?i)([@~+%&]?)([a-z_\-\[\]\\^{}|`+"`"+`][a-z0-9_\-\[\]\\^{}|`+"`"+`]*)$`)

	p.db.Exec("INSERT INTO nicks VALUES ($1, $2, $3)", nick, channel, modes)
}
