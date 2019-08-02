package extra

import (
	"html"
	"net/url"

	"github.com/lrstanley/girc"

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

func newGooglePlugin(c *girc.Client) {
	c.Handlers.AddBg(seabird.PrefixCommand("g"), googleWebCallback)
	c.Handlers.AddBg(seabird.PrefixCommand("gi"), googleImageCallback)

	/*
		cm.Event("g", googleWebCallback, &seabird.HelpInfo{
			Usage:       "<query>",
			Description: "Retrieves top Google web search result for given query",
		})

		cm.Event("gi", googleImageCallback, &seabird.HelpInfo{
			Usage:       "<query>",
			Description: "Retrieves top Google images search result for given query",
		})
	*/
}

func googleWebCallback(c *girc.Client, e girc.Event) {
	googleSearch(c, e, "web", e.Last())
}

func googleImageCallback(c *girc.Client, e girc.Event) {
	googleSearch(c, e, "images", e.Last())
}

func googleSearch(c *girc.Client, e girc.Event, service, query string) {
	if query == "" {
		c.Cmd.ReplyTof(e, "Query required")
		return
	}

	gr := &googleResponse{}
	err := utils.GetJSON(
		"https://ajax.googleapis.com/ajax/services/search/"+service+"?v=1.0&q="+url.QueryEscape(query),
		gr)

	if err != nil {
		c.Cmd.ReplyTof(e, "%s", err)
		return
	}

	if len(gr.ResponseData.Results) == 0 {
		c.Cmd.ReplyTof(e, "Error fetching search results")
		return
	}

	c.Cmd.ReplyTof(e, "%s: %s", html.UnescapeString(gr.ResponseData.Results[0].Title), gr.ResponseData.Results[0].URL)
}
