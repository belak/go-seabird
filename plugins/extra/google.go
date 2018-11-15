package extra

import (
	"html"
	"net/url"

	seabird "github.com/belak/go-seabird"
	"github.com/belak/go-seabird/plugins/utils"
	irc "gopkg.in/irc.v3"
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

func googleWebCallback(b *seabird.Bot, m *irc.Message) {
	googleSearch(b, m, "web", m.Trailing())
}

func googleImageCallback(b *seabird.Bot, m *irc.Message) {
	googleSearch(b, m, "images", m.Trailing())
}

func googleSearch(b *seabird.Bot, m *irc.Message, service, query string) {
	go func() {
		if query == "" {
			b.MentionReply(m, "Query required")
			return
		}

		gr := &googleResponse{}
		err := utils.GetJSON(
			"https://ajax.googleapis.com/ajax/services/search/"+service+"?v=1.0&q="+url.QueryEscape(m.Trailing()),
			gr)

		if err != nil {
			b.MentionReply(m, "%s", err)
			return
		}

		if len(gr.ResponseData.Results) == 0 {
			b.MentionReply(m, "Error fetching search results")
			return
		}

		b.MentionReply(m, "%s: %s", html.UnescapeString(gr.ResponseData.Results[0].Title), gr.ResponseData.Results[0].URL)
	}()
}
