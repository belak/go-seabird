package plugins

import (
	seabird ".."

	"encoding/json"
)

type DownPlugin struct {
	Bot *seabird.Bot
}

func init() {
	seabird.RegisterPlugin("down", NewChancePlugin)
}

func NewDownPlugin(b *seabird.Bot, c json.RawMessage) {
	p := &ChancePlugin{Bot: b}

	/*
		err := json.Unmarshal(c, &p.Key)
		if err != nil {
			fmt.Println(err)
		}
	*/

	b.RegisterFunction("coin", p.CoinKick)
	//b.RegisterFunction("roulette", p.ForecastCurrent)
}
