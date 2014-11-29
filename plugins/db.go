package plugins

import (
	"github.com/belak/seabird/bot"
	"github.com/jmoiron/sqlx"
)

// Note: We do not use database/sql because this allows for a number of simplifications

func init() {
	bot.RegisterPlugin("db", NewDBPlugin)
}

type DBPlugin struct {
	Driver     string
	DataSource string
}

func NewDBPlugin(b *bot.Bot) (bot.Plugin, *sqlx.DB, error) {
	p := &DBPlugin{}

	err := b.Config("db", p)
	if err != nil {
		return nil, nil, err
	}

	db, err := sqlx.Connect(p.Driver, p.DataSource)

	return p, db, err
}
