package plugins

import (
	"net/http"
	"bytes"
	json "github.com/bitly/go-simplejson"
	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
)

func init() {
	bot.RegisterPlugin("tiny", NewTinyPlugin)
}

func NewTinyPlugin(m *mux.CommandMux) error {
	m.Event("tiny", Shorten)

	return nil
}

func Shorten(c *irc.Client, e *irc.Event) {
	url := "https://www.googleapis.com/urlshortener/v1/url"

	var jsonStr = []byte(`{"longUrl":"` + e.Trailing() + `"}`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := json.NewFromReader(resp.Body)
	if err != nil {
		c.MentionReply(e, "Error reading response from server")
		return
	}

	id, err := body.Get("id").String()
	if err != nil {
		c.MentionReply(e, "Error reading JSON data")
		return
	}

	c.MentionReply(e, id)
}
