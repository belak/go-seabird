package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

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
	if len(os.Args) < 2 {
		log.Fatalln("Not enough args. Please pass at least a server name.")
	}

	// Command line options (just in case)
	flag.Parse()

	// Connect to mongo
	mgo_url, err := url.Parse(os.Getenv("MONGO_HOST"))
	if err != nil {
		fmt.Println(err)
		return
	}
	mgo_url.Scheme = ""
	sess, err := mgo.Dial(mgo_url.String())
	if err != nil {
		fmt.Println(err)
		return
	}

	// Import/export config
	if os.Args[1] == "import" {
		if len(os.Args) < 2 {
			log.Fatalf("usage: %s import [filename]\n", os.Args[0])
		}

		data := &Config{}

		// Open the file
		file, err := os.Open(os.Args[2])
		defer file.Close()

		// Read the json
		r := json.NewDecoder(file)
		err = r.Decode(data)
		if err != nil {
			log.Fatalln(err)
		}

		for _, v := range data.Plugins {
			if v2, ok := v["pluginname"]; !ok {
				fmt.Println(v)
				log.Fatalln("At least one plugin config does not contain a pluginname")
			} else {
				if _, ok := v2.(string); !ok {
					log.Fatalln("At least one plugin config contain a pluginname that is not a string")
				}
			}
		}

		fmt.Println(data.Client.ConnectionName)
		_, err = sess.DB("seabird").C("seabird").Upsert(bson.M{"connectionname": data.Client.ConnectionName}, data.Client)
		if err != nil {
			log.Fatalln(err)
		}

		for _, v := range data.Plugins {
			name := v["pluginname"].(string)
			_, err = sess.DB("seabird").C("config").Upsert(bson.M{"pluginname": name}, v)
			if err != nil {
				log.Fatalln(err)
			}
		}
	} else if os.Args[1] == "export" {
		if len(os.Args) < 3 {
			log.Fatalf("usage: %s export [server name]\n", os.Args[0])
		}

		data := &bot.ClientConfig{}
		err = sess.DB("seabird").C("seabird").Find(bson.M{"connectionname": os.Args[2]}).One(data)
		if err != nil {
			log.Fatalln(err)
		}

		var moreData []map[string]interface{}
		err = sess.DB("seabird").C("config").Find(nil).All(&moreData)
		if err != nil {
			log.Fatalln(err)
		}

		// Remove the internal mongo IDs
		for k := range moreData {
			delete(moreData[k], "_id")
		}

		dataOut := &Config{
			data,
			moreData,
		}

		out, err := json.MarshalIndent(dataOut, "", "\t")
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Println(string(out))
	} else {
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
}
