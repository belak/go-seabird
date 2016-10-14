package main

import (
	"os"

	"github.com/Sirupsen/logrus"

	// Load plugins
	_ "github.com/belak/go-seabird/plugins"
	_ "github.com/belak/go-seabird/plugins/extra"
	_ "github.com/belak/go-seabird/plugins/url"

	// Load the core
	"github.com/belak/go-seabird"
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
