package extra

import (
	"fmt"
	"net/url"
	"strings"

	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/plugins/utils"
	irc "gopkg.in/irc.v3"
)

func init() {
	seabird.RegisterPlugin("stock", newStockPlugin)
}

type stockPrice struct {
	Price float64 `json:"price"`
}

func newStockPlugin(cm *seabird.CommandMux) {
	cm.Event("stock", search, &seabird.HelpInfo{
		Usage:       "<symbol>",
		Description: "Gets current stock price for the given symbol",
	})
}

func search(b *seabird.Bot, m *irc.Message) {
	go func() {
		if m.Trailing() == "" {
			b.MentionReply(m, "Symbol required")
			return
		}

		symbols := strings.Split(strings.ToUpper(m.Trailing()), " ")

		url := "https://api.iextrading.com/1.0/stock/market/batch?types=price&symbols=" + url.QueryEscape(strings.Join(symbols, ","))

		response := make(map[string]stockPrice)
		err := utils.GetJSON(url, &response)
		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}

		var prices []string
		for _, symbol := range symbols {
			prices = append(prices, fmt.Sprintf("%s: %.2f", symbol, response[symbol].Price))
		}

		b.MentionReply(m, "%s", strings.Join(prices, ", "))
	}()
}
