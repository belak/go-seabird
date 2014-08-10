package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"labix.org/v2/mgo"

	"bitbucket.org/belak/seabird"
	"bitbucket.org/belak/seabird/auth"
	"bitbucket.org/belak/seabird/bot"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("Not enough args. Please pass a server name.")
	}

	// Command line options (just in case)
	flag.Parse()

	// Firstly, make a list of all possible plugin factories
	bot.RegisterPlugin("chance", seabird.NewChancePlugin)
	bot.RegisterPlugin("dice", seabird.NewDicePlugin)
	bot.RegisterPlugin("forecast", seabird.NewForecastPlugin)
	bot.RegisterPlugin("karma", seabird.NewKarmaPlugin)
	bot.RegisterPlugin("mentions", seabird.NewMentionsPlugin)
	bot.RegisterPlugin("url", seabird.NewURLPlugin)

	// Now all the auth plugins
	bot.RegisterAuthPlugin("generic", auth.NewGenericAuthPlugin)

	// Connect to mongo
	sess, err := mgo.Dial("localhost")
	if err != nil {
		fmt.Println(err)
		return
	}

	// Make the bot
	b, err := bot.NewBot(sess, os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}

	err = b.Run()

	if err != nil {
		log.Fatalln(err)
	}
}
