package main

import (
	"os"

	// Load plugins
	"github.com/Sirupsen/logrus"
	_ "github.com/belak/go-seabird/plugins"
	_ "github.com/belak/go-seabird/plugins/linkproviders"
	"github.com/belak/go-seabird/seabird"

	// Load DB drivers
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func failIfErr(err error, desc string) {
	if err != nil {
		logrus.WithError(err).Fatalln(desc)
	}
}

func main() {
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
	err = b.Run()
	failIfErr(err, "Failed to create run bot")
}
