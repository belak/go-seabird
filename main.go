package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"

	"labix.org/v2/mgo"

	"github.com/belak/seabird/bot"

	// Load plugins
	_ "github.com/belak/seabird/auth"
	_ "github.com/belak/seabird/plugins"
)

type Config struct {
	Client  *bot.ClientConfig
	Plugins []map[string]interface{}
}

func main() {
	// Command line options (just in case)
	flag.Parse()

	// Connect to mongo
	mgo_url, err := url.Parse(os.Getenv("MONGO_HOST"))
	if err != nil {
		fmt.Println(err)
		return
	}
	mgo_url.Scheme = "mongodb"
	sess, err := mgo.Dial(mgo_url.String())
	if err != nil {
		fmt.Println(err)
		return
	}

	// Make the bot
	b, err := bot.NewBot(sess, os.Getenv("SEABIRD_NETWORK"))
	if err != nil {
		log.Fatalln(err)
	}

	err = b.Run()

	if err != nil {
		log.Fatalln(err)
	}
}
