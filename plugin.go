package seabird

import "github.com/belak/go-plugin"

var plugins = plugin.NewRegistry()

// RegisterPlugin registers a PluginFactory for a given name. It will
// panic if multiple plugins are registered with the same name.
func RegisterPlugin(name string, factory interface{}) {
	err := plugins.Register(name, factory)
	if err != nil {
		panic(err.Error())
	}
}
