package extra

import (
	"context"

	"github.com/go-xorm/xorm"
	"xorm.io/core"

	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/internal"
)

func init() {
	seabird.RegisterPlugin("db", NewDBPlugin)
}

const contextKeyDB = internal.ContextKey("seabird-db")

func CtxDB(ctx context.Context) *xorm.Engine {
	return ctx.Value(contextKeyDB).(*xorm.Engine)
}

type dbConfig struct {
	Driver      string
	DataSource  string
	TablePrefix string
}

// NewDBPlugin instantiates a new database connection from a bot with a valid
// db config section.
func NewDBPlugin(b *seabird.Bot) error {
	dbc := &dbConfig{}

	err := b.Config("db", dbc)
	if err != nil {
		return err
	}

	engine, err := xorm.NewEngine(dbc.Driver, dbc.DataSource)
	if err != nil {
		return err
	}

	// Ensure table and column mapping is set up how we want it. This means
	// using the GonicMapper as a base (so stuff like ID is converted properly)
	// but also adding a table prefix (if set) and caching the results (similar
	// to the default mapper).
	var (
		columnMapper core.IMapper = core.NewCacheMapper(core.GonicMapper{})
		tableMapper  core.IMapper = core.GonicMapper{}
	)

	if dbc.TablePrefix != "" {
		tableMapper = core.NewPrefixMapper(tableMapper, dbc.TablePrefix)
	}

	tableMapper = core.NewCacheMapper(tableMapper)

	engine.SetColumnMapper(columnMapper)
	engine.SetTableMapper(tableMapper)

	b.SetValue(contextKeyDB, engine)

	return nil
}
