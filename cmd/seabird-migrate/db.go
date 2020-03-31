package main

import (
	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/plugins/extra/db"
	"github.com/belak/nut"

	"xorm.io/xorm"
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

	err := db.NewDBPlugin(b)
	if err != nil {
		return nil, nil, err
	}

	xdb := db.CtxDB(b.Context())

	ndb, err := nut.Open(dbc.Filename, 0700)
	if err != nil {
		return nil, nil, err
	}

	return ndb, xdb, nil
}
