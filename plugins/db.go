package plugins

import (
	"github.com/jmoiron/sqlx"

	"github.com/belak/seabird/bot"
)

func init() {
	bot.RegisterPlugin("db", NewDBPlugin)
}

type dbConfig struct {
	Driver     string
	DataSource string
}

func NewDBPlugin(b *bot.Bot) (bot.Plugin, error) {
	dbc := &dbConfig{}
	err := b.Config("db", dbc)
	if err != nil {
		return nil, err
	}

	db, err := sqlx.Connect(dbc.Driver, dbc.DataSource)
	if err != nil {
		return nil, err
	}

	return db, nil
}
