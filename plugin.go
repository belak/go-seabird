package seabird

import (
	"fmt"

	"github.com/gobwas/glob"
	"github.com/unknwon/com"
)

type PluginFactory func(b *Bot) error

var plugins = map[string]PluginFactory{}

// RegisterPlugin registers a PluginFactory for a given name. It will
// panic if multiple plugins are registered with the same name.
func RegisterPlugin(name string, factory PluginFactory) {
	if _, ok := plugins[name]; ok {
		panic(fmt.Sprintf("Plugin %q registered multiple times", name))
	}

	plugins[name] = factory
}

func matchingPlugins(rawWhitelist, rawBlacklist []string) ([]string, error) {
	var whitelist []glob.Glob
	var blacklist []glob.Glob

	// Compile all of the whitelist into globs
	for _, rawGlob := range rawWhitelist {
		g, err := glob.Compile(rawGlob, '.')
		if err != nil {
			return nil, err
		}
		whitelist = append(whitelist, g)
	}

	// Compile all of the blacklist into globs
	for _, rawGlob := range rawBlacklist {
		g, err := glob.Compile(rawGlob, '.')
		if err != nil {
			return nil, err
		}
		blacklist = append(blacklist, g)
	}

	// If the whitelist is empty, we want to match all plugins.
	if len(rawWhitelist) == 0 {
		whitelist = append(whitelist, glob.MustCompile("**", '.'))
	}

	var matching []string
	for item := range plugins {
		if matchesGloblist(item, whitelist) && !matchesGloblist(item, blacklist) {
			matching = com.AppendStr(matching, item)
		}
	}

	return matching, nil
}

// matchesGloblist is a simple function which tries an item against a
// slice of globs. It returns true if any of them match.
func matchesGloblist(item string, list []glob.Glob) bool {
	for _, glob := range list {
		if glob.Match(item) {
			return true
		}
	}

	return false
}
