package bot

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/codegangsta/inject"
)

func init() {
	// Grab the type of the interfaces we need
	var e error
	var p Plugin

	errorInterfaceType = inject.InterfaceOf(&e)
	pluginInterfaceType = inject.InterfaceOf(&p)
}

var errorInterfaceType reflect.Type
var pluginInterfaceType reflect.Type

type pluginLoadStatus struct {
	loaded loadStatus

	// Maps each node to the types it requires (in set form)
	requires map[reflect.Type]struct{}

	// Maps each node to a list of types it provides
	provides []reflect.Type
}

type loadStatus int

const (
	unloaded loadStatus = iota
	loading
	loaded
)

func newPluginLoadStatus() *pluginLoadStatus {
	return &pluginLoadStatus{
		unloaded,
		make(map[reflect.Type]struct{}),
		nil,
	}
}

func (b *Bot) determineLoadOrder() ([]string, error) {
	// This is essentially a queue of "next up" plugins - they've had all their deps satisfied
	var input []string

	// This is the order plugins should be loaded in
	var output []string

	// Map each plugin to its status
	pluginStatus := make(map[string]*pluginLoadStatus)

	// Maps each type to the plugin that provides it
	providedBy := make(map[reflect.Type]string)

	// Check each plugin
	for _, v := range b.config.Plugins {
		if _, ok := pluginStatus[v]; ok {
			return nil, fmt.Errorf("Plugin '%s' cannot be loaded more than once.", v)
		}

		if _, ok := plugins[v]; !ok {
			return nil, fmt.Errorf("Plugin '%s' cannot be found.", v)
		}

		// Get the plugin constructor and do some validation
		con := plugins[v]
		t := reflect.TypeOf(con)
		if t.Kind() != reflect.Func {
			return nil, fmt.Errorf("Plugin '%s' has a constructor which is not a function", v)
		}

		if t.NumOut() < 2 {
			return nil, fmt.Errorf("Plugin '%s' has a constructor which doesn't return enough values", v)
		}

		if t.Out(0) != pluginInterfaceType {
			return nil, fmt.Errorf("Plugin '%s' has a constructor which doesn't contain an Plugin as the first return")
		}

		if t.Out(t.NumOut()-1) != errorInterfaceType {
			return nil, fmt.Errorf("Plugin '%s' has a constructor which doesn't contain an error as the last return")
		}

		s := newPluginLoadStatus()

		// Loop through all input and add it to required
		// NOTE: We skip anything already in the Injector so we can have a common core.
		for i := 0; i < t.NumIn(); i++ {
			if !b.inj.Get(t.In(i)).IsValid() {
				s.requires[t.In(i)] = struct{}{}
			}
		}

		// If there weren't any requirements, it's safe to load
		if len(s.requires) == 0 {
			input = append(input, v)
		}

		// Loop through all output and add it to provided
		// NOTE: We skip the first and last values, as those are supposed to be the plugin itself and the error
		for i := 1; i < t.NumOut()-1; i++ {
			s.provides = append(s.provides, t.Out(i))
			if _, ok := providedBy[t.Out(i)]; ok {
				return nil, fmt.Errorf("Type '%s' is provided by multiple plugins.", t.Out(i))
			}
			providedBy[t.Out(i)] = v
		}

		pluginStatus[v] = s
	}

	// Now that we know about all the plugins, we can load them in order
	for len(input) > 0 {
		// Pop the last element off of input and add it to the output
		lastIdx := len(input) - 1
		n := input[lastIdx]
		input = input[:lastIdx]
		output = append(output, n)
		pluginStatus[n].loaded = loaded

		// Loop through all the things the plugin we just popped off provides
		for _, r := range pluginStatus[n].provides {
			// Loop through all leftover nodes
			for k, n := range pluginStatus {
				// If it hasn't been added to the list yet
				if n.loaded == unloaded {
					// If it's in the requirements, we can remove it
					delete(n.requires, r)

					// If there are no more requirements, throw it into the list
					if len(n.requires) == 0 {
						input = append(input, k)
						n.loaded = loading
					}
				}
			}
		}

		fmt.Println("Added plugin", n)
	}

	// At this point all the plugins that could be loaded have been loaded.
	// The only reason for a plugin to not be in the list is for a dep to
	// not be satisfied.
	var errs []string
	for n, p := range pluginStatus {
		if p.loaded == unloaded {
			for r := range p.requires {
				if _, ok := providedBy[r]; ok {
					errs = append(errs, fmt.Sprintf("%s needs %s (provided by %s)", n, r, providedBy[r]))
				} else {
					errs = append(errs, fmt.Sprintf("%s needs %s", n, r))
				}
			}
		}
	}

	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, ", "))
	}

	return output, nil
}

// TODO: More error checking
func (b *Bot) loadPlugin(name string) error {
	p, ok := plugins[name]
	if !ok {
		return fmt.Errorf("Plugin '%s' does not exist.", name)
	}

	vals, err := b.inj.Invoke(p)
	if err != nil {
		return err
	}

	if len(vals) < 2 {
		return fmt.Errorf("Plugin '%s' did not return enough values.", name)
	}

	// Grab the plugin, the error, and cut them out of vals
	plugin := vals[0]
	reflectedErr := vals[len(vals)-1]
	vals = vals[1 : len(vals)-1]

	b.plugins[name] = plugin.Interface()

	if err, ok = reflectedErr.Interface().(error); err != nil && !ok {
		return fmt.Errorf("Plugin '%s' did not return an error as the last value.", name)
	}

	if err != nil {
		return err
	}

	for _, v := range vals {
		b.inj.Set(v.Type(), v)
	}

	return nil
}
