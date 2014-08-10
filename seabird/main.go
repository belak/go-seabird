package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"labix.org/v2/mgo"

	"bitbucket.org/belak/seabird/bot"

	// Load plugins
	_ "bitbucket.org/belak/seabird"
	_ "bitbucket.org/belak/seabird/auth"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("Not enough args. Please pass a server name.")
	}

	// Command line options (just in case)
	flag.Parse()

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
