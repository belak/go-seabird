package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path"

	"labix.org/v2/mgo"

	"bitbucket.org/belak/irc"
	"bitbucket.org/belak/irc/mux"
	"bitbucket.org/belak/seabird"
	"bitbucket.org/belak/seabird/auth"
)

type Config struct {
	Prefix string

	// Bot info
	Nick string
	User string
	Name string
	Pass string

	// Host
	Host        string
	TLS         bool
	TLSNoVerify bool

	// Cmds for on connect
	Cmds []string

	// Plugin config
	Plugins struct {
		Forecast string
	}

	AuthSalt string
}

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

	flag.StringVar(&configFile, "F", path.Join(home, ".seabird", "main.json"), "alternate config file")
}

var configFile string

func loadConfig(filename string) *Config {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	config := &Config{}
	dec := json.NewDecoder(file)
	dec.Decode(config)

	return config
}

func main() {
	// Command line options
	flag.Parse()

	// Load the config file
	config := loadConfig(configFile)

	c := irc.NewClient(config.Nick, config.User, config.Name, config.Pass)

	// Connect to mongo
	sess, err := mgo.Dial("localhost")
	if err != nil {
		fmt.Println(err)
		return
	}

	db := sess.DB("seabird")
	au := auth.NewGenericAuth(c, db, config.Prefix, config.AuthSalt)

	// Add seabird
	cmds := mux.NewCommandMux(config.Prefix)
	ment := mux.NewMentionMux()

	// Coin Kick
	cmds.ChannelFunc("coin", seabird.CoinKickHandler)

	// URL stuff
	c.EventFunc("PRIVMSG", seabird.URLHandler)

	// Dice rolling
	ment.EventFunc(seabird.DiceHandler)

	// Mentions
	ment.EventFunc(seabird.MentionsHandler)

	// Add karma
	k := seabird.NewKarmaHandler(db.C("karma"))
	cmds.EventFunc("karma", k.Karma)
	c.EventFunc("PRIVMSG", k.Msg)

	// Add forecast
	f := seabird.NewForecastHandler(config.Plugins.Forecast, db.C("weather"))
	cmds.Event("*", f)

	// Add say
	cmds.PrivateFunc("say", au.CheckPermFunc("seabird.say", seabird.SayHandler))

	// channops
	cmds.EventFunc("join", au.CheckPermFunc("seabird.join", seabird.JoinHandler))
	cmds.ChannelFunc("part", au.CheckPermFunc("seabird.part", seabird.PartHandler))

	// Add our muxes to the bot
	c.Event("PRIVMSG", cmds)
	c.Event("PRIVMSG", ment)

	// Things to do on connect
	c.EventFunc("001", func(c *irc.Client, e *irc.Event) {
		for _, v := range config.Cmds {
			c.Write(v)
		}
	})

	if config.TLS {
		// Have to work around self signed ssl cert
		conf := &tls.Config{
			InsecureSkipVerify: config.TLSNoVerify,
		}

		err = c.DialTLS(config.Host, conf)
	} else {
		err = c.Dial(config.Host)
	}

	if err != nil {
		fmt.Println(err)
	}
}
