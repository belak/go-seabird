package plugins

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/jinzhu/gorm"

	"github.com/belak/go-seabird/seabird"
	"github.com/belak/irc"
)

func init() {
	seabird.RegisterPlugin("karma", newKarmaPlugin)
}

type karmaPlugin struct {
	db *gorm.DB
}

type KarmaTarget struct {
	gorm.Model
	Name  string `gorm:"unique_index"`
	Score int
}

var regex = regexp.MustCompile(`([^\s]+)(\+\+|--)(?:\s|$)`)

func newKarmaPlugin(b *seabird.Bot, m *seabird.BasicMux, cm *seabird.CommandMux, db *gorm.DB) {
	p := &karmaPlugin{db: db}

	p.db.AutoMigrate(&KarmaTarget{})

	cm.Event("karma", p.karmaCallback, &seabird.HelpInfo{
		Usage:       "<nick>",
		Description: "Gives karma for given user",
	})

	cm.Event("topkarma", p.topKarmaCallback, &seabird.HelpInfo{
		Description: "Reports the user with the most karma",
	})

	cm.Event("bottomkarma", p.bottomKarmaCallback, &seabird.HelpInfo{
		Description: "Reports the user with the least karma",
	})

	m.Event("PRIVMSG", p.callback)
}

func (p *karmaPlugin) cleanedName(name string) string {
	return strings.TrimFunc(strings.ToLower(name), unicode.IsSpace)
}

// GetKarmaFor returns the karma for the given name.
func (p *karmaPlugin) GetKarmaFor(name string) int {
	out := &KarmaTarget{}
	p.db.Where(&KarmaTarget{Name: p.cleanedName(name)}).First(&out)
	return out.Score
}

// UpdateKarma will update the karma for a given name and return the new karma value.
func (p *karmaPlugin) UpdateKarma(name string, diff int) int {
	target := &KarmaTarget{Name: p.cleanedName(name)}

	tx := p.db.Begin()
	defer tx.Commit()

	p.db.FirstOrCreate(target, target)

	target.Score += diff

	p.db.Save(target)

	return target.Score
}

func (p *karmaPlugin) karmaCallback(b *seabird.Bot, m *irc.Message) {
	term := strings.TrimSpace(m.Trailing())

	// If we don't provide a term, search for the current nick
	if term == "" {
		term = m.Prefix.Name
	}

	b.MentionReply(m, "%s's karma is %d", term, p.GetKarmaFor(term))
}

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
