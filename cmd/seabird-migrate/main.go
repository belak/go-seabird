package main // import "github.com/belak/go-seabird/cmd/seabird"

import (
	"math/rand"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	// Officially supported DB drivers
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	// Load plugins
	//_ "github.com/belak/go-seabird-bucket"
	//_ "github.com/belak/go-seabird-uno"
	_ "github.com/belak/go-seabird/plugins"
	_ "github.com/belak/go-seabird/plugins/extra"
	_ "github.com/belak/go-seabird/plugins/url"

	// Load the core
	seabird "github.com/belak/go-seabird"
)

func failIfErr(err error, desc string) {
	if err != nil {
		logrus.WithError(err).Fatalln(desc)
	}
}

func main() {
	// Seed the random number generator for plugins to use.
	rand.Seed(time.Now().UTC().UnixNano())

	conf := os.Getenv("SEABIRD_CONFIG")
	if conf == "" {
		conf = "config.toml"
		_, err := os.Stat(conf)
		failIfErr(err, "Failed to load config")
	}

	confReader, err := os.Open(conf)
	failIfErr(err, "Failed to load config")

	// Create the bot
	b, err := seabird.NewBot(confReader)
	failIfErr(err, "Failed to create new bot")

	// Load the relevant databases
	nutdb, xormdb, err := openDBs(b)
	failIfErr(err, "Failed to open databases")

	// Migrate karma
	err = migrateKarma(b, nutdb, xormdb)
	failIfErr(err, "Failed to migrate karma")

	// Migrate phrases
	err = migratePhrases(b, nutdb, xormdb)
	failIfErr(err, "Failed to migrate phrases")
}
