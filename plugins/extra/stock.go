package extra

import (
	"fmt"
	"strings"

	iex "github.com/goinvest/iexcloud"

	seabird "github.com/belak/go-seabird"
	irc "gopkg.in/irc.v3"
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
	err := b.Config("stock", p)
	if err != nil {
		return err
	}

	p.Client = iex.NewClient(p.Key, stockBaseURL)

	cm.Event("stock", p.search, &seabird.HelpInfo{
		Usage:       "<symbol>",
		Description: "Gets current stock price for the given symbol",
	})

	return nil
}

func (p *stockPlugin) search(b *seabird.Bot, m *irc.Message) {
	go func() {
		if m.Trailing() == "" {
			b.MentionReply(m, "Symbol required")
			return
		}

		symbols := strings.Split(strings.ToUpper(m.Trailing()), " ")
		prices := []string{}

		for _, symbol := range symbols {
			price, err := p.Client.Price(symbol)
			if err != nil {
				b.MentionReply(m, "%s", err)
				continue
			}
			prices = append(prices, fmt.Sprintf("%s: %.2f", symbol, price))
		}

		b.MentionReply(m, "%s", strings.Join(prices, ", "))
	}()
}
