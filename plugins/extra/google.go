package extra

import (
	"html"
	"net/url"

	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/plugins/utils"
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

func newGooglePlugin(cm *seabird.CommandMux) {
	cm.Event("g", googleWebCallback, &seabird.HelpInfo{
		Usage:       "<query>",
		Description: "Retrieves top Google web search result for given query",
	})

	cm.Event("gi", googleImageCallback, &seabird.HelpInfo{
		Usage:       "<query>",
		Description: "Retrieves top Google images search result for given query",
	})
}

func googleWebCallback(b *seabird.Bot, r *seabird.Request) {
	googleSearch(b, r, "web", r.Message.Trailing())
}

func googleImageCallback(b *seabird.Bot, r *seabird.Request) {
	googleSearch(b, r, "images", r.Message.Trailing())
}

func googleSearch(b *seabird.Bot, r *seabird.Request, service, query string) {
	go func() {
		if query == "" {
			r.MentionReply("Query required")
			return
		}

		gr := &googleResponse{}
		err := utils.GetJSON(
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
