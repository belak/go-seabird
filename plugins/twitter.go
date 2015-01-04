package plugins

import (
	"github.com/ChimeraCoder/anaconda"
	"github.com/belak/seabird/bot"
)

func init() {
	bot.RegisterPlugin("twitter", NewTwitterPlugin)
}

type TwitterPlugin struct {
	ConsumerKey       string
	ConsumerSecret    string
	AccessToken       string
	AccessTokenSecret string
}

func NewTwitterPlugin(b *bot.Bot) (*anaconda.TwitterApi, error) {
	p := &TwitterPlugin{}

	err := b.Config("twitter", p)
	if err != nil {
		return nil, err
	}

	// TODO: Check connection
	anaconda.SetConsumerKey(p.ConsumerKey)
	anaconda.SetConsumerSecret(p.ConsumerSecret)
	api := anaconda.NewTwitterApi(p.AccessToken, p.AccessTokenSecret)

	return api, nil
}
