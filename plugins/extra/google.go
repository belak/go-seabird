package extra

import (
	"html"
	"net/url"

	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/internal"
)

func init() {
	seabird.RegisterPlugin("google", newGooglePlugin)
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

func newGooglePlugin(b *seabird.Bot) error {
	cm := b.CommandMux()

	cm.Event("g", googleWebCallback, &seabird.HelpInfo{
		Usage:       "<query>",
		Description: "Retrieves top Google web search result for given query",
	})

	cm.Event("gi", googleImageCallback, &seabird.HelpInfo{
		Usage:       "<query>",
		Description: "Retrieves top Google images search result for given query",
	})

	return nil
}

func googleWebCallback(r *seabird.Request) {
	googleSearch(r, "web", r.Message.Trailing())
}

func googleImageCallback(r *seabird.Request) {
	googleSearch(r, "images", r.Message.Trailing())
}

func googleSearch(r *seabird.Request, service, query string) {
	go func() {
		if query == "" {
			r.MentionReply("Query required")
			return
		}

		gr := &googleResponse{}
		err := internal.GetJSON(
			"https://ajax.googleapis.com/ajax/services/search/"+service+"?v=1.0&q="+url.QueryEscape(query),
			gr)

		if err != nil {
			r.MentionReply("%s", err)
			return
		}

		if len(gr.ResponseData.Results) == 0 {
			r.MentionReply("Error fetching search results")
			return
		}

		r.MentionReply("%s: %s", html.UnescapeString(gr.ResponseData.Results[0].Title), gr.ResponseData.Results[0].URL)
	}()
}
