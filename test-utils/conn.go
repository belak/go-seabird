package utils

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-irc/irc"
	"github.com/stretchr/testify/assert"
)

// TestClientServer is a simple abstraction meant to be used as an
// io.ReadWriter with seabird.Bot so messages can be tracked.
type TestClientServer struct {
	client *bytes.Buffer
	server *bytes.Buffer
}

// NewTestClientServer returns a new TestClientServer
func NewTestClientServer() *TestClientServer {
	return &TestClientServer{
		client: &bytes.Buffer{},
		server: &bytes.Buffer{},
	}
}

// Read is what will be coming from the "server"
func (cs *TestClientServer) Read(p []byte) (int, error) {
	return cs.server.Read(p)
}

// Write is what will be going to the "server"
func (cs *TestClientServer) Write(p []byte) (int, error) {
	return cs.client.Write(p)
}

// SendServerLines will send all given irc.Messages as if they were
// coming from the server, to be read by the test client.
func (cs *TestClientServer) SendServerLines(lines []string) {
	// We're writing as if it's coming from the server
	w := irc.NewWriter(cs.server)

	for _, line := range lines {
		w.WriteMessage(irc.MustParseMessage(line))
	}
}

// CheckLines will ensure that the client sent all the expected lines
// (and nothing more) and return true if they did and false if they
// didn't.
func (cs *TestClientServer) CheckLines(t *testing.T, expected []string) bool {
	ok := true

	// Split all the lines
	lines := strings.Split(cs.client.String(), "\r\n")
	//lines := strings.Split(strings.TrimRight(cs.client.String(), "\r\n"), "\r\n")

	// Loop through all the expected lines
	var line, clientLine string
	for len(expected) > 0 && len(lines) > 0 {
		// Pop off the next expected and incoming lines
		line, expected = expected[0], expected[1:]
		clientLine, lines = lines[0], lines[1:]

		ok = ok && assert.Equal(t, line, clientLine)
	}

	// Ensure all the expected and incoming lines were used up
	ok = ok && assert.Equal(t, 0, len(expected), "Not enough lines: %s", strings.Join(expected, ", "))
	ok = ok && assert.Equal(t, 0, len(lines), "Extra non-empty lines: %s", strings.Join(lines, ", "))

	return ok
}

// Reset clears the contents of the internal buffers
func (cs *TestClientServer) Reset() {
	cs.client.Reset()
	cs.server.Reset()
}
