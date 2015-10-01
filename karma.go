package plugins

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/jmoiron/sqlx"
)

func init() {
	bot.RegisterPlugin("karma", NewKarmaPlugin)
}

type karmaUser struct {
	Name  string
	Score int
}

type KarmaPlugin struct {
	db *sqlx.DB
}

var regex = regexp.MustCompile(`([^\s]+)(\+\+|--)(?:\s|$)`)

func NewKarmaPlugin(b *bot.Bot) (bot.Plugin, error) {
	b.LoadPlugin("db")
	p := &KarmaPlugin{b.Plugins["db"].(*sqlx.DB)}

	b.CommandMux.Event("karma", p.karmaCallback, &bot.HelpInfo{
		Usage:       "<nick>",
		Description: "Gives karma for given user",
	})
	b.CommandMux.Event("topkarma", p.topKarmaCallback, &bot.HelpInfo{
		Description: "Reports the user with the most karma",
	})
	b.CommandMux.Event("bottomkarma", p.bottomKarmaCallback, &bot.HelpInfo{
		Description: "Reports the user with the least karma",
	})
	b.BasicMux.Event("PRIVMSG", p.callback)

	return p, nil
}

func (p *KarmaPlugin) cleanedName(name string) string {
	return strings.TrimFunc(strings.ToLower(name), unicode.IsSpace)
}

// GetKarmaFor returns the karma for the given name.
func (p *KarmaPlugin) GetKarmaFor(name string) int {
	var score int
	err := p.db.Get(&score, "SELECT score FROM karma WHERE name=$1", p.cleanedName(name))
	if err != nil {
		return 0
	}

	return score
}

// UpdateKarma will update the karma for a given name and return the new karma value.
func (p *KarmaPlugin) UpdateKarma(name string, diff int) int {
	_, err := p.db.Exec("INSERT INTO karma (name, score) VALUES ($1, $2)", p.cleanedName(name), diff)
	// If it was a nil error, we got the insert
	if err == nil {
		return diff
	}

	// Grab a transaction, just in case
	tx, err := p.db.Beginx()
	defer tx.Commit()

	if err != nil {
		fmt.Println("TX:", err)
	}

	// If there was an error, we try an update.
	_, err = tx.Exec("UPDATE karma SET score=score+$1 WHERE name=$2", diff, p.cleanedName(name))
	if err != nil {
		fmt.Println("UPDATE:", err)
	}

	var score int
	err = tx.Get(&score, "SELECT score FROM karma WHERE name=$1", p.cleanedName(name))
	if err != nil {
		fmt.Println("SELECT:", err)
	}

	return score
}

func (p *KarmaPlugin) karmaCallback(b *bot.Bot, m *irc.Message) {
	term := strings.TrimSpace(m.Trailing())

	// If we don't provide a term, search for the current nick
	if term == "" {
		term = m.Prefix.Name
	}

	b.MentionReply(m, "%s's karma is %d", term, p.GetKarmaFor(term))
}

func (p *KarmaPlugin) topKarmaCallback(b *bot.Bot, m *irc.Message) {
	user := &karmaUser{}
	err := p.db.Get(user, "SELECT name, score FROM karma ORDER BY score DESC LIMIT 1")
	if err != nil {
		b.MentionReply(m, "Error fetching scores")
		return
	}

	b.MentionReply(m, "%s has the top karma with %d", user.Name, user.Score)
}

func (p *KarmaPlugin) bottomKarmaCallback(b *bot.Bot, m *irc.Message) {
	user := &karmaUser{}
	err := p.db.Get(user, "SELECT name, score FROM karma ORDER BY score ASC LIMIT 1")
	if err != nil {
		b.MentionReply(m, "Error fetching scores")
		return
	}

	b.MentionReply(m, "%s has the bottom karma with %d", user.Name, user.Score)
}

func (p *KarmaPlugin) callback(b *bot.Bot, m *irc.Message) {
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
