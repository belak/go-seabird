package extra

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode"

	"github.com/belak/go-seabird"
	irc "github.com/go-irc/irc/v2"
)

func init() {
	seabird.RegisterPlugin("bulkcnam", newBulkCNAMPlugin)
}

type bulkCNAMPlugin struct {
	Key string
}

func newBulkCNAMPlugin(b *seabird.Bot, cm *seabird.CommandMux) error {
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
func (p *bulkCNAMPlugin) bulkCNAMCallback(b *seabird.Bot, m *irc.Message) {
	number := m.Trailing()

	for _, digit := range number {
		if !unicode.IsDigit(digit) {
			b.MentionReply(m, "Error: Not a phone number")
			return
		}
	}

	resp, err := http.Get(fmt.Sprintf("http://cnam.bulkcnam.com/?id=%s&did=%s", p.Key, number))
	if err != nil {
		b.MentionReply(m, "Error: BulkCNAM appears to be down")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b.MentionReply(m, "Error: Server side error occurred")
		return
	}

	in := bufio.NewReader(resp.Body)
	line, err := in.ReadString('\n')
	if err != nil || err == io.EOF {
		b.MentionReply(m, "%s", strings.TrimSpace(line))
	} else {
		b.MentionReply(m, "Error: Something happened when parsing the response.")
	}
}
