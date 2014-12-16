package bot

// Requirements for PluginFactories are as follows:
// 1. Be a function
// 2. Return at least 1 value
// 3. The last return value is an error
//
// Anything before the last argument will be treated as values this plugin provides.
// Anything this package needs should be taken in as an argument to the constructor.
//
// Unfortunately, PluginFactories are not type safe because of the requirements for
// dependency injection to work, but as plugins are only loaded on startup (and not
// for every event) these should be simple to test.
type PluginFactory interface{}
