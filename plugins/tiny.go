package plugins

import (
	"bytes"
	"encoding/json"
	"github.com/belak/irc"
	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/mux"
	"net/http"
)

type ShortenResult struct {
	Kind    string `json:"kind"`
	Id      string `json:"id"`
	LongUrl string `json:"longUrl"`
}

func init() {
	bot.RegisterPlugin("tiny", NewTinyPlugin)
}

func NewTinyPlugin(m *mux.CommandMux) error {
	m.Event("tiny", Shorten)

	return nil
}

func Shorten(c *irc.Client, e *irc.Event) {
	go func() {
		if e.Trailing() == "" {
			c.MentionReply(e, "URL required")
			return
		}

		url := "https://www.googleapis.com/urlshortener/v1/url"

		data := map[string]string{"longUrl": e.Trailing()}
		out, err := json.Marshal(data)
		var jsonStr = []byte(out)
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.MentionReply(e, "Error connecting to goo.gl")
			return
		}
		defer resp.Body.Close()

		sr := new(ShortenResult)
		err = json.NewDecoder(resp.Body).Decode(sr)
		if err != nil {
			c.MentionReply(e, "Error reading server response")
			return
		}

		c.MentionReply(e, sr.Id)
	}()
}
