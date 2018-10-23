package extra

import (
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"

	"github.com/belak/go-seabird"
	"github.com/belak/nut"
)

func init() {
	seabird.RegisterPlugin("db", newDBPlugin)
}

type dbConfig struct {
	Filename    string
	Driver      string
	DataSource  string
	TablePrefix string
}

func newDBPlugin(b *seabird.Bot) (*nut.DB, *xorm.Engine, error) {
	dbc := &dbConfig{}
	err := b.Config("db", dbc)
	if err != nil {
		return nil, nil, err
	}

	// We want to default to nil if there's no Filename, but we still need to
	// keep it around for migrating data.
	var ndb *nut.DB

	if dbc.Filename != "" {
		ndb, err = nut.Open(dbc.Filename, 0700)
		if err != nil {
			return nil, nil, err
		}
	}

	engine, err := xorm.NewEngine(dbc.Driver, dbc.DataSource)
	if err != nil {
		return nil, nil, err
	}

	// Ensure table and column mapping is set up how we want it. This means
	// using the GonicMapper as a base (so stuff like ID is converted properly)
	// but also adding a table prefix (if set) and caching the results (similar
	// to the default mapper).
	var columnMapper core.IMapper = core.NewCacheMapper(core.GonicMapper{})
	var tableMapper core.IMapper = core.GonicMapper{}
	if dbc.TablePrefix != "" {
		tableMapper = core.NewPrefixMapper(tableMapper, dbc.TablePrefix)
	}
	tableMapper = core.NewCacheMapper(tableMapper)

	engine.SetColumnMapper(columnMapper)
	engine.SetTableMapper(tableMapper)

	return ndb, engine, nil
}
