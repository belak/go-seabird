package extra

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode"

	seabird "github.com/belak/go-seabird"
)

func init() {
	seabird.RegisterPlugin("bulkcnam", newBulkCNAMPlugin)
}

type bulkCNAMPlugin struct {
	Key string
}

func newBulkCNAMPlugin(b *seabird.Bot) error {
	cm := b.CommandMux()

	p := &bulkCNAMPlugin{}

	err := b.Config("bulkcnam", p)
	if err != nil {
		return err
	}

	cm.Event("cnam", p.bulkCNAMCallback, &seabird.HelpInfo{
		Usage:       "<phone #>",
		Description: "Returns the CNAM of a phone number",
	})

	return nil
}

// This function queries the BulkCNAM API for a Phone #'s
// corresponding CNAM, and returns it
func (p *bulkCNAMPlugin) bulkCNAMCallback(ctx context.Context, r *seabird.Request) {
	number := r.Message.Trailing()

	for _, digit := range number {
		if !unicode.IsDigit(digit) {
			r.MentionReply("Error: Not a phone number")
			return
		}
	}

	resp, err := http.Get(fmt.Sprintf("http://cnam.bulkcnam.com/?id=%s&did=%s", p.Key, number))
	if err != nil {
		r.MentionReply("Error: BulkCNAM appears to be down")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		r.MentionReply("Error: Server side error occurred")
		return
	}

	in := bufio.NewReader(resp.Body)

	line, err := in.ReadString('\n')
	if err != nil || err == io.EOF {
		r.MentionReply("%s", strings.TrimSpace(line))
	} else {
		r.MentionReply("Error: Something happened when parsing the response.")
	}
}
