package extra

import (
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"

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
	Driver      string
	DataSource  string
	TablePrefix string
}

func newDBPlugin(b *seabird.Bot) (*xorm.Engine, error) {
	conf := &dbConfig{}
	err := b.Config("db", conf)
	if err != nil {
		return nil, err
	}

	db, err := xorm.NewEngine(conf.Driver, conf.DataSource)
	if err != nil {
		return nil, err
	}

	// Set table and column mapping rules. We want to use the GonicMapper so we
	// can use normal Go names, but we also want to have the option to add a
	// prefix if people want.
	var mapper core.IMapper = core.GonicMapper{}
	if conf.TablePrefix != "" {
		mapper = core.NewPrefixMapper(mapper, conf.TablePrefix)
	}
	db.SetMapper(mapper)

	return db, nil
}
