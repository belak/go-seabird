package plugins

import (
	"encoding/json"
	"html"
	"net/http"
	"net/url"

	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
)

func init() {
	bot.RegisterPlugin("google", NewGooglePlugin)
}

type GoogleResponse struct {
	ResponseData struct {
		Results  []struct {
			Url string `json:"unescapedUrl"`
			Title string `json:"titleNoFormatting"`
		} `json:"results"`
	} `json:"responseData"`
	ResponseStatus int `json:"responseStatus"`
}

func NewGooglePlugin(c *mux.CommandMux) error {
	c.Event("g", "query", Web)
	c.Event("gi", "query", Image)

	return nil
}

func Web(c * irc.Client, e *irc.Event) {
	googleSearch(c, e, "web", e.Trailing())
}

func Image(c * irc.Client, e *irc.Event) {
	googleSearch(c, e, "images", e.Trailing())
}

func googleSearch(c *irc.Client, e *irc.Event, service, query string) {
	go func() {
		if query == "" {
			c.MentionReply(e, "Query required")
			return
		}

		resp, err := http.Get("https://ajax.googleapis.com/ajax/services/search/" + service + "?v=1.0&q=" + url.QueryEscape(e.Trailing()))
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}
		defer resp.Body.Close()

		gr := &GoogleResponse{}
		err = json.NewDecoder(resp.Body).Decode(gr)
		if err != nil {
			c.MentionReply(e, "%s", err)
			return
		}

		if gr.ResponseStatus != 200 || len(gr.ResponseData.Results) == 0 {
			c.MentionReply(e, "Error fetching search results")
			return
		}

		c.MentionReply(e, "%s: %s", html.UnescapeString(gr.ResponseData.Results[0].Title), gr.ResponseData.Results[0].Url)
	}()
}
