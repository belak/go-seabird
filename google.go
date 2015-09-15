package plugins

import (
	"encoding/json"
	"html"
	"net/http"
	"net/url"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
)

func init() {
	bot.RegisterPlugin("google", NewGooglePlugin)
}

type googleResponse struct {
	ResponseData struct {
		Results []struct {
			URL   string `json:"unescapedUrl"`
			Title string `json:"titleNoFormatting"`
		} `json:"results"`
	} `json:"responseData"`
	ResponseStatus int `json:"responseStatus"`
}

func NewGooglePlugin(b *bot.Bot) (bot.Plugin, error) {
	b.CommandMux.Event("g", googleWebCallback, &bot.HelpInfo{
		Usage:       "<query>",
		Description: "Retrieves top Google web search result for given query",
	})
	b.CommandMux.Event("gi", googleImageCallback, &bot.HelpInfo{
		Usage:       "<query>",
		Description: "Retrieves top Google images search result for given query",
	})

	return nil, nil
}

func googleWebCallback(b *bot.Bot, m *irc.Message) {
	googleSearch(b, m, "web", m.Trailing())
}

func googleImageCallback(b *bot.Bot, m *irc.Message) {
	googleSearch(b, m, "images", m.Trailing())
}

func googleSearch(b *bot.Bot, m *irc.Message, service, query string) {
	go func() {
		if query == "" {
			b.MentionReply(m, "Query required")
			return
		}

		resp, err := http.Get("https://ajax.googleapis.com/ajax/services/search/" + service + "?v=1.0&q=" + url.QueryEscape(m.Trailing()))
		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}
		defer resp.Body.Close()

		gr := &googleResponse{}
		err = json.NewDecoder(resp.Body).Decode(gr)
		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}

		if gr.ResponseStatus != 200 || len(gr.ResponseData.Results) == 0 {
			b.MentionReply(m, "Error fetching search results")
			return
		}

		b.MentionReply(m, "%s: %s", html.UnescapeString(gr.ResponseData.Results[0].Title), gr.ResponseData.Results[0].URL)
	}()
}
