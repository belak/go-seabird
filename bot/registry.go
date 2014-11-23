package bot

import (
	"errors"
	"fmt"
)

func init() {
	plugins = make(map[string]PluginFactory)
}

var plugins map[string]PluginFactory

func RegisterPlugin(name string, p PluginFactory) error {
	if _, ok := plugins[name]; ok {
		return errors.New(fmt.Sprintf("There is already a plugin named '%s' registered.", name))
	}

	// TODO: Log for real
	fmt.Printf("Plugin '%s' registered.\n", name)

	plugins[name] = p

	return nil
}
