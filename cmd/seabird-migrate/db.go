package main

import (
	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/plugins/extra"
	"github.com/belak/nut"

	"github.com/go-xorm/xorm"
)

type dbConfig struct {
	// Filename comes from the original version of the plugin and is needed to
	// load nutdb.
	Filename string
}

func openDBs(b *seabird.Bot) (*nut.DB, *xorm.Engine, error) {
	dbc := &dbConfig{}

	if err := b.Config("db", dbc); err != nil {
		return nil, nil, err
	}

	xdb, err := extra.NewDBPlugin(b)
	if err != nil {
		return nil, nil, err
	}

	ndb, err := nut.Open(dbc.Filename, 0700)
	if err != nil {
		return nil, nil, err
	}

	return ndb, xdb, nil
}
