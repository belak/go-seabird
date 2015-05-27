package plugins

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/jmoiron/sqlx"

	"github.com/belak/seabird/bot"
	"github.com/belak/sorcix-irc"
)

type KarmaUser struct {
	Name  string
	Score int
}

type KarmaPlugin struct {
	db *sqlx.DB
}

var regex = regexp.MustCompile(`((?:\w+[\+-]?)*\w)(\+\+|--)(?:\s|$)`)

func NewKarmaPlugin(db *sqlx.DB) bot.Plugin {
	return &KarmaPlugin{
		db,
	}
}

func (p *KarmaPlugin) Register(b *bot.Bot) error {
	b.CommandMux.Event("karma", p.Karma, &bot.HelpInfo{
		"<nick>",
		"Gives karma for given user",
	})
	b.CommandMux.Event("topkarma", p.TopKarma, &bot.HelpInfo{
		Description: "Reports the user with the most karma",
	})
	b.CommandMux.Event("bottomkarma", p.BottomKarma, &bot.HelpInfo{
		Description: "Reports the user with the least karma",
	})
	b.BasicMux.Event("PRIVMSG", p.Msg)

	return nil
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

func (p *KarmaPlugin) Karma(b *bot.Bot, m *irc.Message) {
	term := strings.TrimSpace(m.Trailing())

	// If we don't provide a term, search for the current nick
	if term == "" {
		term = m.Prefix.Name
	}

	b.MentionReply(m, "%s's karma is %d", term, p.GetKarmaFor(term))
}

func (p *KarmaPlugin) TopKarma(b *bot.Bot, m *irc.Message) {
	user := &KarmaUser{}
	err := p.db.Get(user, "SELECT name, score FROM karma ORDER BY score DESC LIMIT 1")
	if err != nil {
		b.MentionReply(m, "Error fetching scores")
		return
	}

	b.MentionReply(m, "%s has the top karma with %d", user.Name, user.Score)
}

func (p *KarmaPlugin) BottomKarma(b *bot.Bot, m *irc.Message) {
	user := &KarmaUser{}
	err := p.db.Get(user, "SELECT name, score FROM karma ORDER BY score ASC LIMIT 1")
	if err != nil {
		b.MentionReply(m, "Error fetching scores")
		return
	}

	b.MentionReply(m, "%s has the bottom karma with %d", user.Name, user.Score)
}

func (p *KarmaPlugin) Msg(b *bot.Bot, m *irc.Message) {
	if len(m.Params) < 2 || !bot.MessageFromChannel(m) {
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
