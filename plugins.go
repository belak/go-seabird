package seabird

import (
	"encoding/json"
	"errors"
	"github.com/thoj/go-ircevent"
)

func init() {
	auth_plugins = make(map[string]RegisterAuthFunc)
	plugins = make(map[string]RegisterCallback)
}

type AuthPlugin interface {
	// Permission checking
	UserCan(user *User, perm string) bool
}

type NilAuthPlugin struct{}

func NewNilAuthPlugin(b *Bot) AuthPlugin {
	return &NilAuthPlugin{}
}

func (p *NilAuthPlugin) UserCan(user *User, perm string) bool {
	return true
}

var plugins map[string]RegisterCallback
var auth_plugins map[string]RegisterAuthFunc

func RegisterPlugin(name string, plugin RegisterCallback) error {
	if _, ok := plugins[name]; ok {
		return errors.New("plugin " + name + " is already registered")
	}

	plugins[name] = plugin

	return nil
}

func RegisterAuthPlugin(name string, plugin RegisterAuthFunc) error {
	if _, ok := auth_plugins[name]; ok {
		return errors.New("auth plugin " + name + " is already registered")
	}

	auth_plugins[name] = plugin

	return nil
}

// Function type for registering a plugin
type RegisterCallback func(b *Bot, c json.RawMessage)
type RegisterAuthFunc func(b *Bot, c json.RawMessage) AuthPlugin

// Types of callbacks
type Callback func(m *irc.Event)

func (b *Bot) RegisterCallback(name string, callback Callback) {
	b.Conn.AddCallback(name, callback)
}

func (b *Bot) RegisterMention(callback Callback) error {
	b.MentionCommands = append(b.MentionCommands, callback)
	return nil
}

func (b *Bot) RegisterFunction(name string, callback Callback) error {
	if _, ok := b.Commands[name]; ok {
		return errors.New("function " + name + " already exists")
	}

	b.Commands[name] = callback

	return nil
}
