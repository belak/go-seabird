package bot

// This is simply so we can store plugins with a name
type Plugin interface{}

// Requirements for PluginFactories are as follows:
// 1. Be a function
// 2. Return at least 2 values
// 3. The first return value is the plugin
// 4. The last return value is an error
//
// Anything between the first and last arguments will be treated as things this plugin provides.
// Anything this package needs should be taken in as an argument to the constructor.
type PluginFactory interface{}

type PluginConfig interface{}
