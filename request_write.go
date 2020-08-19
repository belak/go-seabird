package seabird

import (
	"errors"
	"fmt"
	"strings"

	irc "gopkg.in/irc.v3"
)

// Reply to a Request with a convenience wrapper around fmt.Sprintf.
func (r *Request) Replyf(format string, v ...interface{}) error {
	if len(r.Message.Params) < 1 || len(r.Message.Params[0]) < 1 {
		return errors.New("Invalid IRC message")
	}

	target := r.Message.Prefix.Name
	if r.FromChannel() {
		target = r.Message.Params[0]
	}

	fullMsg := fmt.Sprintf(format, v...)
	for _, resp := range strings.Split(fullMsg, "\n") {
		r.WriteMessage(&irc.Message{
			Prefix:  &irc.Prefix{},
			Command: "PRIVMSG",
			Params: []string{
				target,
				resp,
			},
		})
	}

	return nil
}

// MentionReply acts the same as Bot.Reply but it will prefix it with the user's
// nick if we are in a channel.
func (r *Request) MentionReplyf(format string, v ...interface{}) error {
	if len(r.Message.Params) < 1 || len(r.Message.Params[0]) < 1 {
		return errors.New("Invalid IRC message")
	}

	target := r.Message.Prefix.Name
	prefix := ""

	if r.FromChannel() {
		target = r.Message.Params[0]
		prefix = r.Message.Prefix.Name + ": "
	}

	fullMsg := fmt.Sprintf(format, v...)
	for _, resp := range strings.Split(fullMsg, "\n") {
		r.WriteMessage(&irc.Message{
			Prefix:  &irc.Prefix{},
			Command: "PRIVMSG",
			Params: []string{
				target,
				prefix + resp,
			},
		})
	}

	return nil
}

// PrivateReply is similar to Reply, but it will always send privately.
func (r *Request) PrivateReplyf(format string, v ...interface{}) {
	r.WriteMessage(&irc.Message{
		Prefix:  &irc.Prefix{},
		Command: "PRIVMSG",
		Params: []string{
			r.Message.Prefix.Name,
			fmt.Sprintf(format, v...),
		},
	})
}

// CTCPReply is a convenience function to respond to CTCP requests.
func (r *Request) CTCPReplyf(format string, v ...interface{}) error {
	if r.Message.Command != "CTCP" {
		return errors.New("Invalid CTCP message")
	}

	r.WriteMessage(&irc.Message{
		Prefix:  &irc.Prefix{},
		Command: "NOTICE",
		Params: []string{
			r.Message.Prefix.Name,
			fmt.Sprintf(format, v...),
		},
	})

	return nil
}

// Send is a simple function to send an IRC event.
func (r *Request) WriteMessage(m *irc.Message) {
	r.bot.WriteMessage(m)
}

// Write will write an raw IRC message to the stream.
func (r *Request) Write(line string) {
	r.bot.Write(line)
}

// Writef is a convenience method around fmt.Sprintf and Bot.Write.
func (r *Request) Writef(format string, args ...interface{}) {
	r.bot.Writef(format, args...)
}
