package plugins

import (
	"github.com/jmoiron/sqlx"

	"github.com/belak/go-seabird/seabird"
)

func init() {
	seabird.RegisterPlugin("db", newDBPlugin)
}

type dbConfig struct {
	Driver     string
	DataSource string
}

func newDBPlugin(b *seabird.Bot) (*sqlx.DB, error) {
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
