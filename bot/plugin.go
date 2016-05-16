package bot

import (
	"fmt"
	"log"
)

func init() {
	plugins = make(map[string]PluginFactory)
}

var plugins map[string]PluginFactory

// Plugin can be any type. It is unfortunate we have to use an empty interface
// here, but in order to allow the user to store whatever they want this needs
// to be the case.
type Plugin interface{}

// PluginFactory is what actually gets registered as a Plugin. It takes a bot
// and returns the plugin (or nil) and an error.
type PluginFactory func(b *Bot) (Plugin, error)

// RegisterPlugin registers a PluginFactory for a given name.
func RegisterPlugin(name string, factory PluginFactory) error {
	if _, ok := plugins[name]; ok {
		log.Fatalln(fmt.Sprintf("Plugin with name %q is already registered", name))
	}

	plugins[name] = factory
	return nil
}
