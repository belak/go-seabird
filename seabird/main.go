package main

import (
	_ "../auth"
	_ "../plugins"

	"../../seabird"

	"encoding/json"
	"flag"
	"log"
	"os"
	"os/user"
	"path"
)

func init() {
	// Try HOME first then fall back to user.Current() because it needs cgo
	home := os.Getenv("HOME")
	if home == "" {
		user, err := user.Current()
		if err != nil {
			log.Fatalln(err)
		}
		home = user.HomeDir
	}

	flag.StringVar(&configFile, "F", path.Join(home, ".seabird", "main.json"), "alternate config file to use")
}

var configFile string

func loadConfig(filename string) *seabird.Config {
	file, err := os.Open(configFile)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	config := &seabird.Config{}
	dec := json.NewDecoder(file)
	dec.Decode(config)

	return config
}

func main() {
	// Command line options
	flag.Parse()

	config := loadConfig(configFile)

	// Get everything ready
	bot, err := seabird.NewBot(config)
	if err != nil {
		log.Println(err)
		return
	}

	// Loop until we actually quit
	bot.Loop()
}
