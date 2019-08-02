package extra

import (
	"fmt"
	"strings"

	iex "github.com/goinvest/iexcloud"
	"github.com/lrstanley/girc"

	seabird "github.com/belak/go-seabird"
)

func init() {
	seabird.RegisterPlugin("stock", newStockPlugin)
}

const stockBaseURL = "https://cloud.iexapis.com/v1/"

type stockPlugin struct {
	Key    string
	Client *iex.Client
}

func newStockPlugin(b *seabird.Bot, c *girc.Client) error {
	p := &stockPlugin{}
	err := b.Config("stock", p)
	if err != nil {
		return err
	}

	p.Client = iex.NewClient(p.Key, stockBaseURL)

	c.Handlers.AddBg(seabird.PrefixCommand("stock"), p.search)

	/*
	cm.Event("stock", p.search, &seabird.HelpInfo{
		Usage:       "<symbol>",
		Description: "Gets current stock price for the given symbol",
	})
	*/

	return nil
}

func (p *stockPlugin) search(c *girc.Client, e girc.Event) {
	if e.Last() == "" {
		c.Cmd.ReplyTof(e, "Symbol required")
		return
	}

	symbols := strings.Split(strings.ToUpper(e.Last()), " ")
	prices := []string{}

	for _, symbol := range symbols {
		price, err := p.Client.Price(symbol)
		if err != nil {
			c.Cmd.ReplyTof(e, "%s", err)
			continue
		}
		prices = append(prices, fmt.Sprintf("%s: %.2f", symbol, price))
	}

	c.Cmd.ReplyTof(e, "%s", strings.Join(prices, ", "))
}
