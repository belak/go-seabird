package extra

import (
	"fmt"
	"strings"

	iex "github.com/goinvest/iexcloud"

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

func newStockPlugin(b *seabird.Bot, cm *seabird.CommandMux) error {
	p := &stockPlugin{}

	if err := b.Config("stock", p); err != nil {
		return err
	}

	p.Client = iex.NewClient(p.Key, stockBaseURL)

	cm.Event("stock", p.search, &seabird.HelpInfo{
		Usage:       "<symbol>",
		Description: "Gets current stock price for the given symbol",
	})

	return nil
}

func (p *stockPlugin) search(b *seabird.Bot, r *seabird.Request) {
	go func() {
		if r.Message.Trailing() == "" {
			b.MentionReply(r, "Symbol required")
			return
		}

		symbols := strings.Split(strings.ToUpper(r.Message.Trailing()), " ")
		prices := []string{}

		for _, symbol := range symbols {
			price, err := p.Client.Price(symbol)
			if err != nil {
				b.MentionReply(r, "%s", err)
				continue
			}
			prices = append(prices, fmt.Sprintf("%s: %.2f", symbol, price))
		}

		b.MentionReply(r, "%s", strings.Join(prices, ", "))
	}()
}
