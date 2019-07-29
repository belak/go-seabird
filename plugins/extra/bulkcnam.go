package extra

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode"

	"github.com/lrstanley/girc"

	seabird "github.com/belak/go-seabird"
)

func init() {
	seabird.RegisterPlugin("bulkcnam", newBulkCNAMPlugin)
}

type bulkCNAMPlugin struct {
	Key string
}

func newBulkCNAMPlugin(b *seabird.Bot, c *girc.Client) error {
	p := &bulkCNAMPlugin{}

	err := b.Config("bulkcnam", p)
	if err != nil {
		return err
	}

	/*
		cm.Event("cnam", p.bulkCNAMCallback, &seabird.HelpInfo{
			Usage:       "<phone #>",
			Description: "Returns the CNAM of a phone number",
		})
	*/

	c.Handlers.AddBg(seabird.PrefixCommand("cnam"), p.bulkCNAMCallback)

	return nil
}

// This function queries the BulkCNAM API for a Phone #'s
// corresponding CNAM, and returns it
func (p *bulkCNAMPlugin) bulkCNAMCallback(c *girc.Client, e girc.Event) {
	number := e.Last()

	for _, digit := range number {
		if !unicode.IsDigit(digit) {
			c.Cmd.ReplyTo(e, "Error: Not a phone number")
			return
		}
	}

	resp, err := http.Get(fmt.Sprintf("http://cnam.bulkcnam.com/?id=%s&did=%s", p.Key, number))
	if err != nil {
		c.Cmd.ReplyTo(e, "Error: BulkCNAM appears to be down")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		c.Cmd.ReplyTo(e, "Error: Server side error occurred")
		return
	}

	in := bufio.NewReader(resp.Body)
	line, err := in.ReadString('\n')
	if err != nil || err == io.EOF {
		c.Cmd.ReplyTof(e, "%s", strings.TrimSpace(line))
	} else {
		c.Cmd.ReplyTo(e, "Error: Something happened when parsing the response.")
	}
}
