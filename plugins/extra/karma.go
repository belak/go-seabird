package extra

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/belak/go-seabird"
	"github.com/belak/nut"
	"github.com/go-irc/irc"
	"github.com/go-xorm/xorm"
)

func init() {
	seabird.RegisterPlugin("karma", newKarmaPlugin)
}

type karmaPlugin struct {
	db *xorm.Engine
}

// karmaTarget represents an item with a karma count
type karmaTarget struct {
	Name  string
	Score int
}

var regex = regexp.MustCompile(`([^\s]+)(\+\+|--)(?:\s|$)`)

func newKarmaPlugin(b *seabird.Bot, m *seabird.BasicMux, cm *seabird.CommandMux, oldDB *nut.DB, db *xorm.Engine) error {
	p := &karmaPlugin{db: db}

	err := p.db.Sync(&karmaTarget{})
	if err != nil {
		return err
	}

	cm.Event("karma", p.karmaCallback, &seabird.HelpInfo{
		Usage:       "<nick>",
		Description: "Gives karma for given user",
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
	out := &karmaTarget{Name: p.cleanedName(name)}

	// We can safely ignore errors here because we specifically want to fall
	// back to the default value.
	p.db.Get(out)

	return out.Score
}

// UpdateKarma will update the karma for a given name and return the new karma value.
func (p *karmaPlugin) UpdateKarma(name string, diff int) int {
	target := &karmaTarget{Name: p.cleanedName(name)}
	out := &karmaTarget{Name: p.cleanedName(name)}

	sess := p.db.NewSession()
	err := sess.Begin()
	if err != nil {
		return 0
	}
	defer sess.Commit()

	found, _ := sess.Get(out)
	if err != nil {
		return 0
	}
	out.Score += diff

	if found {
		sess.Update(out, target)
	} else {
		sess.Insert(out)
	}

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
	res := &karmaTarget{}
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
	if len(m.Params) < 2 || !m.FromChannel() {
		return
	}

	matches := regex.FindAllStringSubmatch(m.Trailing(), -1)
	if len(matches) > 0 {
		for _, v := range matches {
			if len(v) < 3 {
				continue
			}

			var diff int
			if v[2] == "++" {
				diff = 1
			} else {
				diff = -1
			}

			name := strings.ToLower(v[1])
			if name == m.Prefix.Name {
				// penalize self-karma
				diff = -1
			}

			b.Reply(m, "%s's karma is now %d", v[1], p.UpdateKarma(name, diff))
		}
	}
}
