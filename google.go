package plugins

import (
	"encoding/json"
	"html"
	"net/http"
	"net/url"

	"github.com/belak/seabird/bot"
	"github.com/belak/irc"
)

func init() {
	bot.RegisterPlugin("google", NewGooglePlugin)
}

type GooglePlugin struct{}

type GoogleResponse struct {
	ResponseData struct {
		Results []struct {
			URL   string `json:"unescapedUrl"`
			Title string `json:"titleNoFormatting"`
		} `json:"results"`
	} `json:"responseData"`
	ResponseStatus int `json:"responseStatus"`
}

func NewGooglePlugin(b *bot.Bot) (bot.Plugin, error) {
	p := &GooglePlugin{}

	b.CommandMux.Event("g", Web, &bot.HelpInfo{
		"<query>",
		"Retrieves top Google web search result for given query",
	})
	b.CommandMux.Event("gi", Image, &bot.HelpInfo{
		"<query>",
		"Retrieves top Google images search result for given query",
	})

	return p, nil
}

func Web(b *bot.Bot, m *irc.Message) {
	googleSearch(b, m, "web", m.Trailing())
}

func Image(b *bot.Bot, m *irc.Message) {
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

		gr := &GoogleResponse{}
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
