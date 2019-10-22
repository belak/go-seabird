package extra

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/go-xorm/xorm"

	seabird "github.com/belak/go-seabird"
	irc "gopkg.in/irc.v3"
)

func init() {
	seabird.RegisterPlugin("karma", newKarmaPlugin)
}

type karmaPlugin struct {
	db *xorm.Engine
}

// Karma represents an item with a karma count
type Karma struct {
	ID    int64
	Name  string `xorm:"unique"`
	Score int
}

var karmaRegex = regexp.MustCompile(`([\w]{2,}|".+?")(\+\++|--+)(?:\s|$)`)

func newKarmaPlugin(b *seabird.Bot, m *seabird.BasicMux, cm *seabird.CommandMux, db *xorm.Engine) error {
	p := &karmaPlugin{db: db}

	// Migrate any relevant tables
	err := db.Sync(Karma{})
	if err != nil {
		return err
	}

	cm.Event("karma", p.karmaCallback, &seabird.HelpInfo{
		Usage:       "<nick>",
		Description: "Displays karma for given user",
	})

	/*
		cm.Event("topkarma", p.topKarmaCallback, &seabird.HelpInfo{
			Description: "Reports the user with the most karma",
		})

		cm.Event("bottomkarma", p.bottomKarmaCallback, &seabird.HelpInfo{
			Description: "Reports the user with the least karma",
		})
	*/

	m.Event("PRIVMSG", p.callback)

	return nil
}

func (p *karmaPlugin) cleanedName(name string) string {
	return strings.TrimFunc(strings.ToLower(name), unicode.IsSpace)
}

// GetKarmaFor returns the karma for the given name.
func (p *karmaPlugin) GetKarmaFor(name string) int {
	out := &Karma{Name: p.cleanedName(name)}

	// Note that we're explicitly ignoring an error here because it's not a
	// problem when this returns zero results.
	_, _ = p.db.Get(out)

	return out.Score
}

// UpdateKarma will update the karma for a given name and return the new karma value.
func (p *karmaPlugin) UpdateKarma(name string, diff int) int {
	out := &Karma{Name: p.cleanedName(name)}

	p.db.Transaction(func(s *xorm.Session) (interface{}, error) {
		found, _ := s.Get(out)
		if !found {
			s.Insert(out)
		}
		out.Score += diff
		return s.ID(out.ID).Cols("score").Update(out)
	})

	return out.Score
}

func (p *karmaPlugin) karmaCallback(b *seabird.Bot, m *irc.Message) {
	term := strings.TrimSpace(m.Trailing())

	// If we don't provide a term, search for the current nick
	if term == "" {
		term = m.Prefix.Name
	}

	b.MentionReply(m, "%s's karma is %d", term, p.GetKarmaFor(term))
}

/*
func (p *karmaPlugin) karmaCheck(b *seabird.Bot, m *irc.Message, msg string, sort string) {
	res := &KarmaTarget{}
	p.db.Order("score " + sort).First(res)
	b.MentionReply(m, "%s has the %s karma with %d", res.Name, msg, res.Score)
}
func (p *karmaPlugin) topKarmaCallback(b *seabird.Bot, m *irc.Message) {
	p.karmaCheck(b, m, "top", "DESC")
}

func (p *karmaPlugin) bottomKarmaCallback(b *seabird.Bot, m *irc.Message) {
	p.karmaCheck(b, m, "bottom", "ASC")
}
*/

func (p *karmaPlugin) callback(b *seabird.Bot, m *irc.Message) {
	if len(m.Params) < 2 || !b.FromChannel(m) {
		return
	}

	var (
		buzzkillTriggered bool
		jerkModeTriggered bool

		changes = make(map[string]int)
	)

	matches := karmaRegex.FindAllStringSubmatch(m.Trailing(), -1)
	for _, v := range matches {
		// If it starts with a ", we know it also ends with a quote so we
		// can chop them off.
		if strings.HasPrefix(v[1], "\"") {
			v[1] = v[1][1 : len(v[1])-1]
		}

		diff := len(v[2]) - 1
		cleanedName := p.cleanedName(v[1])
		cleanedNick := p.cleanedName(m.Prefix.Name)

		// If it's negative, or positive and someone is trying to change
		// their own karma we need to reverse the sign.
		if v[2][0] == '-' || cleanedName == cleanedNick {
			diff *= -1
		}

		changes[v[1]] += diff
	}

	for name, diff := range changes {
		if diff > 5 {
			buzzkillTriggered = true
			diff = 5
		}

		if diff < -5 {
			jerkModeTriggered = true
			diff = -5
		}

		b.Reply(m, "%s's karma is now %d", name, p.UpdateKarma(name, diff))
	}

	if buzzkillTriggered {
		b.Reply(m, "Buzzkill Mode (tm) enforced a maximum karma change of 5")
	}

	if jerkModeTriggered {
		b.Reply(m, "Don't Be a Jerk Mode (tm) enforced a maximum karma change of 5")
	}
}
