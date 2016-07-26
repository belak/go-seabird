package main

import (
	"log"
	"os"

	// Load plugins
	_ "github.com/belak/go-seabird/plugins"
	_ "github.com/belak/go-seabird/plugins/linkproviders"
	"github.com/belak/go-seabird/seabird"

	// Load DB drivers
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	conf := os.Getenv("SEABIRD_CONFIG")
	if conf == "" {
		conf = "config.toml"
		_, err := os.Stat(conf)
		if os.IsNotExist(err) {
			log.Fatalln("$SEABIRD_CONFIG is not defined and config.toml doesn't exist")
		} else if err != nil {
			log.Fatalln(err)
		}
	}

	confReader, err := os.Open(conf)
	if err != nil {
		log.Fatalln(err)
	}

	// Create the bot
	b, err := seabird.NewBot(confReader)
	if err != nil {
		log.Fatalln(err)
	}

	// Run the bot
	err = b.Run()
	if err != nil {
		log.Fatalln(err)
	}
}
