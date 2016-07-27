package seabird

import "github.com/belak/go-plugin"

var plugins = plugin.NewRegistry()

// RegisterPlugin registers a PluginFactory for a given name.
func RegisterPlugin(name string, factory interface{}) error {
	return plugins.Register(name, factory)
}
