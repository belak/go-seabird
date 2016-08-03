package plugins

import (
	"github.com/belak/go-seabird/seabird"
	"github.com/belak/nut"
)

func init() {
	seabird.RegisterPlugin("db", newDBPlugin)
}

type dbConfig struct {
	Filename string
}

func newDBPlugin(b *seabird.Bot) (*nut.DB, error) {
	dbc := &dbConfig{}
	err := b.Config("db", dbc)
	if err != nil {
		return nil, err
	}

	ndb, err := nut.Open(dbc.Filename, 0700)
	if err != nil {
		return nil, err
	}

	return ndb, nil
}
