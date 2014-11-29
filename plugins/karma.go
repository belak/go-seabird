package plugins

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/jmoiron/sqlx"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
)

func init() {
	bot.RegisterPlugin("karma", NewKarmaPlugin)
}

type KarmaPlugin struct {
	db *sqlx.DB
}

var regex = regexp.MustCompile(`((?:\w+[\+-]?)*\w)(\+\+|--)(?:\s|$)`)

func NewKarmaPlugin(c *mux.CommandMux, b *irc.BasicMux, db *sqlx.DB) (bot.Plugin, error) {
	p := &KarmaPlugin{
		db,
	}

	c.Event("karma", p.Karma) // "[object]"
	b.Event("PRIVMSG", p.Msg)

	return p, nil
}

func (p *KarmaPlugin) CleanedName(name string) string {
	return strings.TrimFunc(strings.ToLower(name), unicode.IsSpace)
}

func (p *KarmaPlugin) GetKarmaFor(name string) int {
	var score int
	err := p.db.Get(&score, "SELECT score FROM karma WHERE name=$1", p.CleanedName(name))
	if err != nil {
		return 0
	}

	return score
}

func (p *KarmaPlugin) UpdateKarma(name string, diff int) int {
	_, err := p.db.Exec("INSERT INTO karma (name, score) VALUES ($1, $2)", p.CleanedName(name), diff)
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
	_, err = tx.Exec("UPDATE karma SET score=score+$1 WHERE name=$2", diff, p.CleanedName(name))
	if err != nil {
		fmt.Println("UPDATE:", err)
	}

	var score int
	err = tx.Get(&score, "SELECT score FROM karma WHERE name=$1", p.CleanedName(name))
	if err != nil {
		fmt.Println("SELECT:", err)
	}

	return score
}

func (p *KarmaPlugin) Karma(c *irc.Client, e *irc.Event) {
	c.MentionReply(e, "%s's karma is %d", e.Trailing(), p.GetKarmaFor(e.Trailing()))
}

func (p *KarmaPlugin) Msg(c *irc.Client, e *irc.Event) {
	if len(e.Args) < 2 || !e.FromChannel() {
		return
	}

	matches := regex.FindAllStringSubmatch(e.Trailing(), -1)
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
			if name == e.Identity.Nick {
				// penalize self-karma
				diff = -1
			}

			c.Reply(e, "%s's karma is now %d", v[1], p.UpdateKarma(name, diff))
		}
	}
}
