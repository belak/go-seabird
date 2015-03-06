package main

import (
	"log"
	"os"

	"github.com/belak/seabird/bot"
	"github.com/jmoiron/sqlx"

	// Load plugins
	//_ "github.com/belak/seabird/auth"
	"github.com/belak/seabird/plugins"
	"github.com/belak/seabird/plugins/linkproviders"

	// Load DB drivers
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type DBConfig struct {
	Driver     string
	DataSource string
}

func main() {
	conf := os.Getenv("SEABIRD_CONFIG")
	if conf == "" {
		log.Fatalln("$SEABIRD_CONFIG is not defined")
	}

	// Create the bot
	b, err := bot.NewBot(conf)
	if err != nil {
		log.Fatalln(err)
	}

	// We only actually do something if there hasn't been an error yet.
	registerPlugin := func(p bot.Plugin) bot.Plugin {
		if err == nil {
			err = b.RegisterPlugin(p)
		}
		return p
	}

	dbc := &DBConfig{}
	err = b.Config("db", dbc)
	if err != nil {
		log.Fatalln(err)
	}

	db, err := sqlx.Connect(dbc.Driver, dbc.DataSource)
	if err != nil {
		log.Fatalln(err)
	}

	registerPlugin(plugins.NewChancePlugin())
	registerPlugin(plugins.NewCTCPPlugin())
	registerPlugin(plugins.NewDicePlugin())
	registerPlugin(plugins.NewMathPlugin())
	registerPlugin(plugins.NewForecastPlugin(db))
	registerPlugin(plugins.NewIssuesPlugin())
	registerPlugin(plugins.NewGooglePlugin())
	registerPlugin(plugins.NewKarmaPlugin(db))
	registerPlugin(plugins.NewLastSeenPlugin(db))
	registerPlugin(plugins.NewMentionsPlugin())
	registerPlugin(plugins.NewMetarPlugin())
	registerPlugin(plugins.NewNetToolsPlugin())
	//registerPlugin(plugins.NewNickTrackerPlugin())
	registerPlugin(plugins.NewTinyPlugin())
	registerPlugin(plugins.NewTVDBPlugin())
	registerPlugin(plugins.NewWikiPlugin())

	up := registerPlugin(plugins.NewURLPlugin()).(*plugins.URLPlugin)

	if err != nil {
		log.Fatalln(err)
	}

	err = linkproviders.NewGithubProvider(b, up)
	if err != nil {
		log.Fatalln(err)
	}

	err = linkproviders.NewTwitterProvider(b, up)
	if err != nil {
		log.Fatalln(err)
	}

	err = linkproviders.NewRedditProvider(up)
	if err != nil {
		log.Fatalln(err)
	}

	err = linkproviders.NewBitbucketProvider(up)
	if err != nil {
		log.Fatalln(err)
	}

	err = b.Run()
	if err != nil {
		log.Fatalln(err)
	}
}
