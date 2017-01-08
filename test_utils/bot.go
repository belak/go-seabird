package utils

import (
	"bytes"
	"testing"

	seabird "github.com/belak/go-seabird"
	"github.com/stretchr/testify/assert"
)

var baseConfig = `
[core]
nick = "seabird"
user = "seabird_user"
name = "Seabird Bot"
pass = "password"

prefix = "!"
`

var expectedBaseOutput = []string{
	"PASS :password",
	"NICK :seabird",
	"USER seabird_user 0.0.0.0 0.0.0.0 :Seabird Bot",
}

func RunTest(t *testing.T, testConfig string, input, output []string) bool {
	cs, b := NewTestBot(t, testConfig)

	// Send these lines
	cs.SendServerLines(input)

	// Run the bot until EOF
	b.Run(cs)

	// Expect the base output
	var expectedOutput []string
	expectedOutput = append(expectedOutput, expectedBaseOutput...)
	expectedOutput = append(expectedOutput, output...)

	return cs.CheckLines(t, expectedOutput)
}

// NewTestBot will return the TestClientServer and seabird.Bot for use
// in the test.
func NewTestBot(t *testing.T, testConfig string) (*TestClientServer, *seabird.Bot) {
	confReader := bytes.NewBufferString(baseConfig)
	confReader.WriteString(testConfig)

	bot, err := seabird.NewBot(confReader)
	assert.NoError(t, err)

	return NewTestClientServer(), bot
}
