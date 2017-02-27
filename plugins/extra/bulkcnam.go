package extra

import (
	"io"
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/belak/go-seabird"
	"github.com/belak/irc"
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

func (p *bulkCNAMPlugin) bulkCNAMCallback(b *seabird.Bot, m *irc.Message) {
	if !m.FromChannel() {
		return
	}

	r, err := p.BulkCNAM(m.Trailing())
	if err != nil {
		b.MentionReply(m, "Error: %s", err)
		return
	}

	b.MentionReply(m, "%s", r)
}

// This function queries the BulkCNAM API for a Phone #'s
// corresponding CNAM, and returns it
func (p *bulkCNAMPlugin) BulkCNAM(number string) (string, error) {
	for _, digit := range number {
		if !unicode.IsDigit(digit) {
			return "", errors.New("Not a phone number")
		}
	}

	resp, err := http.Get(fmt.Sprintf("http://cnam.bulkcnam.com/?id=%s&did=%s", p.Key, number))
	if err != nil {
		return "", errors.New("BulkCNAM appears to be down")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New("Server side error occurred")
	}

	in := bufio.NewReader(resp.Body)
	for {
		line, err := in.ReadString('\n')
		if err != nil && err != io.EOF {
			break
		}

		return strings.TrimSpace(line), nil
	}

	return "", errors.New("No results")
}
