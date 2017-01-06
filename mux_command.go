package seabird

import (
	"sort"
	"strings"

	"github.com/go-irc/irc"
)

// CommandMux is a simple IRC event multiplexer, based on the BasicMux.

// HelpInfo is a collection of instructions for command usage that
// is formatted with <prefix>help
type HelpInfo struct {
	Usage       string
	Description string
}

// The CommandMux is given a prefix string and matches all PRIVMSG
// events which start with it. The first word after the string is
// moved into the Event.Command.
type CommandMux struct {
	private *BasicMux
	public  *BasicMux
	prefix  string
	cmdHelp map[string]*HelpInfo
}

// NewCommandMux will create an initialized BasicMux with no handlers.
func NewCommandMux(prefix string) *CommandMux {
	m := &CommandMux{
		NewBasicMux(),
		NewBasicMux(),
		prefix,
		make(map[string]*HelpInfo),
	}

	m.Event("help", m.help, &HelpInfo{
		"<command>",
		"Displays help messages for a given command",
	})
	return m
}

func (m *CommandMux) help(b *Bot, msg *irc.Message) {
	cmd := msg.Trailing()
	if cmd == "" {
		// Get all keys
		keys := make([]string, 0, len(m.cmdHelp))
		for k := range m.cmdHelp {
			keys = append(keys, k)
		}

		// Sort everything
		sort.Strings(keys)

		if msg.FromChannel() {
			// If they said "!help" in a channel, list all available commands
			b.Reply(msg, "Available commands: %s. Use %shelp [command] for more info.", strings.Join(keys, ", "), m.prefix)
		} else {
			for _, v := range keys {
				h := m.cmdHelp[v]
				if h.Usage != "" {
					b.Reply(msg, "%s %s: %s", v, h.Usage, h.Description)
				} else {
					b.Reply(msg, "%s: %s", v, h.Description)
				}
			}
		}
	} else if help, ok := m.cmdHelp[cmd]; ok {
		if help == nil {
			b.Reply(msg, "There is no help available for command %q", cmd)
		} else {
			lines := help.format(m.prefix, cmd)
			for _, line := range lines {
				b.Reply(msg, "%s", line)
			}
		}
	} else {
		b.MentionReply(msg, "There is no help available for command %q", cmd)
	}
}

func (h *HelpInfo) format(prefix, command string) []string {
	if h.Usage == "" && h.Description == "" {
		return []string{"There is no help available for command " + command}
	}

	ret := []string{}

	if h.Usage != "" {
		ret = append(ret, "Usage: "+prefix+command+" "+h.Usage)
	}

	if h.Description != "" {
		ret = append(ret, h.Description)
	}

	return ret
}

// Event will register a Handler as both a private and public command
func (m *CommandMux) Event(c string, h HandlerFunc, help *HelpInfo) {
	m.private.Event(c, h)
	m.public.Event(c, h)

	m.cmdHelp[c] = help
}

// Channel will register a handler as a public command
func (m *CommandMux) Channel(c string, h HandlerFunc, help *HelpInfo) {
	m.public.Event(c, h)

	m.cmdHelp[c] = help
}

// Private will register a handler as a private command
func (m *CommandMux) Private(c string, h HandlerFunc, help *HelpInfo) {
	m.private.Event(c, h)

	m.cmdHelp[c] = help
}

// HandleEvent strips off the prefix, pulls the command out
// and runs HandleEvent on the internal BasicMux
func (m *CommandMux) HandleEvent(b *Bot, msg *irc.Message) {
	if msg.Command != "PRIVMSG" {
		// TODO: Log this
		return
	}

	// Get the last arg and see if it starts with the command prefix
	lastArg := msg.Trailing()
	if !strings.HasPrefix(lastArg, m.prefix) {
		return
	}

	// Copy it into a new Event
	newEvent := msg.Copy()

	// Chop off the command itself
	msgParts := strings.SplitN(lastArg, " ", 2)
	newEvent.Params[len(newEvent.Params)-1] = ""
	if len(msgParts) > 1 {
		newEvent.Params[len(newEvent.Params)-1] = strings.TrimSpace(msgParts[1])
	}

	newEvent.Command = msgParts[0][len(m.prefix):]

	if newEvent.FromChannel() {
		m.public.HandleEvent(b, newEvent)
	} else {
		m.private.HandleEvent(b, newEvent)
	}
}
