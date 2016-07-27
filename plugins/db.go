package plugins

import (
	"github.com/jinzhu/gorm"
	"github.com/jmoiron/sqlx"

	"github.com/belak/go-seabird/seabird"
)

func init() {
	seabird.RegisterPlugin("db", newDBPlugin)
}

type dbConfig struct {
	Driver     string
	DataSource string
	Verbose    bool
}

func newDBPlugin(b *seabird.Bot) (*sqlx.DB, *gorm.DB, error) {
	dbc := &dbConfig{}
	err := b.Config("db", dbc)
	if err != nil {
		return nil, nil, err
	}

	db, err := sqlx.Connect(dbc.Driver, dbc.DataSource)
	if err != nil {
		return nil, nil, err
	}

	gdb, err := gorm.Open(dbc.Driver, dbc.DataSource)
	if err != nil {
		return nil, nil, err
	}

	if dbc.Verbose {
		gdb.LogMode(true)
	}

	return db, gdb, nil
}
