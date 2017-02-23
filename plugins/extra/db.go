package extra

import (
	"github.com/jinzhu/gorm"

	"github.com/belak/go-seabird"
	"github.com/belak/nut"
)

func init() {
	seabird.RegisterPlugin("nutdb", newNutDBPlugin)
	seabird.RegisterPlugin("db", newDBPlugin)
}

type nutDBConfig struct {
	Filename string
}

func newNutDBPlugin(b *seabird.Bot) (*nut.DB, error) {
	dbc := &nutDBConfig{}
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

type dbConfig struct {
	Driver     string
	DataSource string
}

func newDBPlugin(b *seabird.Bot) (*gorm.DB, error) {
	conf := &dbConfig{}
	err := b.Config("db", conf)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(conf.Driver, conf.DataSource)
	if err != nil {
		return nil, err
	}

	return db, nil
}
