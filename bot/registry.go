package bot

import "fmt"

func init() {
	plugins = make(map[string]PluginFactory)
}

var plugins map[string]PluginFactory

func RegisterPlugin(name string, p PluginFactory) {
	if _, ok := plugins[name]; ok {
		panic(fmt.Sprintf("There is already a plugin named '%s' registered.", name))
	}

	plugins[name] = p
}
