package bot

import (
	"errors"
	"fmt"
)

func init() {
	plugins = make(map[string]PluginFactory)
	authPlugins = make(map[string]AuthPluginFactory)
}

var plugins map[string]PluginFactory
var authPlugins map[string]AuthPluginFactory

func RegisterPlugin(name string, p PluginFactory) error {
	if _, ok := plugins[name]; ok {
		return errors.New(fmt.Sprintf("There is already a plugin named '%s' registered.", name))
	}

	// TODO: Log for real
	fmt.Printf("Plugin '%s' registered.\n", name)

	plugins[name] = p

	return nil
}

func RegisterAuthPlugin(name string, p AuthPluginFactory) error {
	if _, ok := authPlugins[name]; ok {
		return errors.New(fmt.Sprintf("There is already an auth plugin named '%s' registered.", name))
	}

	// TODO: Log for real
	fmt.Printf("AuthPlugin '%s' registered.\n", name)

	authPlugins[name] = p

	return nil
}
