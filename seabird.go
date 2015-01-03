package main

import (
	"log"

	"github.com/belak/seabird/bot"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"

	// Load plugins
	//_ "github.com/belak/seabird/auth"
	_ "github.com/belak/seabird/plugins"
	_ "github.com/belak/seabird/plugins/link_providers"

	// Load DB drivers
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var configOverride *string = flag.String("config", "", "alternate config name")

func main() {
	// Command line options (just in case)
	flag.Parse()

	if *configOverride != "" {
		viper.SetConfigFile(*configOverride)
	}

	viper.SetConfigName("seabird")
	viper.AddConfigPath("/etc")
	viper.AddConfigPath("$HOME/.config/seabird")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalln(err)
	}

	// Create the bot
	b, err := bot.NewBot()
	if err != nil {
		log.Fatalln(err)
	}

	err = b.Run()
	if err != nil {
		log.Fatalln(err)
	}
}
