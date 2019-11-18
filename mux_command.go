package seabird

import (
	"sort"
	"strings"
)

// CommandMux is a simple IRC event multiplexer, based on the BasicMux.

// HelpInfo is a collection of instructions for command usage that
// is formatted with <prefix>help
type HelpInfo struct {
	name        string
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
		"help",
		"<command>",
		"Displays help messages for a given command",
	})

	return m
}

func (m *CommandMux) help(r *Request) {
	cmd := r.Message.Trailing()
	if cmd == "" {
		// Get all keys
		keys := make([]string, 0, len(m.cmdHelp))
		for k := range m.cmdHelp {
			keys = append(keys, k)
		}

		// Sort everything
		sort.Strings(keys)

		if r.FromChannel() {
			// If they said "!help" in a channel, list all available commands
			r.Reply("Available commands: %s. Use %shelp [command] for more info.", strings.Join(keys, ", "), m.prefix)
		} else {
			for _, v := range keys {
				h := m.cmdHelp[v]
				if h.Usage != "" {
					r.Reply("%s %s: %s", v, h.Usage, h.Description)
				} else {
					r.Reply("%s: %s", v, h.Description)
				}
			}
		}
	} else if help, ok := m.cmdHelp[cmd]; ok {
		if help == nil {
			r.Reply("There is no help available for command %q", cmd)
		} else {
			lines := help.format(m.prefix, cmd)
			for _, line := range lines {
				r.Reply("%s", line)
			}
		}
	} else {
		r.MentionReply("There is no help available for command %q", cmd)
	}
}

func (h *HelpInfo) format(prefix, command string) []string {
	if h.Usage == "" && h.Description == "" {
		return []string{"There is no help available for command " + command}
	}

	ret := []string{}

	if h.Usage != "" {
		ret = append(ret, "Usage: "+prefix+h.name+" "+h.Usage)
	}

	if h.Description != "" {
		ret = append(ret, h.Description)
	}

	return ret
}

// Event will register a Handler as both a private and public command
func (m *CommandMux) Event(c string, h HandlerFunc, help *HelpInfo) {
	if help != nil {
		help.name = c
	}

	c = strings.ToLower(c)

	m.private.Event(c, h)
	m.public.Event(c, h)

	m.cmdHelp[c] = help
}

// Channel will register a handler as a public command
func (m *CommandMux) Channel(c string, h HandlerFunc, help *HelpInfo) {
	if help != nil {
		help.name = c
	}

	c = strings.ToLower(c)

	m.public.Event(c, h)

	m.cmdHelp[c] = help
}

// Private will register a handler as a private command
func (m *CommandMux) Private(c string, h HandlerFunc, help *HelpInfo) {
	if help != nil {
		help.name = c
	}

	c = strings.ToLower(c)

	m.private.Event(c, h)

	m.cmdHelp[c] = help
}

// HandleEvent strips off the prefix, pulls the command out
// and runs HandleEvent on the internal BasicMux
func (m *CommandMux) HandleEvent(r *Request) {
	timer := r.Timer("command_mux")
	defer timer.Done()

	if r.Message.Command != "PRIVMSG" {
		// TODO: Log this
		return
	}

	// Get the last arg and see if it starts with the command prefix
	lastArg := r.Message.Trailing()
	if r.FromChannel() && !strings.HasPrefix(lastArg, m.prefix) {
		return
	}

	// TODO(jsvana): this loses multi-plugin timing information.
	// Copy it into a new Event
	newRequest := r.Copy()

	// Chop off the command itself
	msgParts := strings.SplitN(lastArg, " ", 2)
	newRequest.Message.Params[len(newRequest.Message.Params)-1] = ""

	if len(msgParts) > 1 {
		newRequest.Message.Params[len(newRequest.Message.Params)-1] = strings.TrimSpace(msgParts[1])
	}

	newRequest.Message.Command = strings.ToLower(msgParts[0])
	newRequest.Message.Command = strings.TrimPrefix(newRequest.Message.Command, m.prefix)

	if newRequest.FromChannel() {
		m.public.HandleEvent(newRequest)
	} else {
		m.private.HandleEvent(newRequest)
	}
}
