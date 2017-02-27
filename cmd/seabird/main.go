package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/Sirupsen/logrus"

	// Load plugins
	_ "github.com/belak/go-seabird/plugins"
	_ "github.com/belak/go-seabird/plugins/extra"
	_ "github.com/belak/go-seabird/plugins/uno"
	_ "github.com/belak/go-seabird/plugins/url"

	// Load DB drivers we care about. We only officially support postgres and
	// sqlite but this should work with any db driver supported by xorm.
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	// Load the core
	"github.com/belak/go-seabird"
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

	// Run the bot
	err = b.ConnectAndRun()
	failIfErr(err, "Failed to create run bot")
}
