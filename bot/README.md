# seabird/bot

This package is the core of the seabird binary.

## Plugins

`bot.RegisterPlugin` takes an `interface{}`, however that empty interface must:

* Be a function
* Return at least 1 value
* The last return value is an error

All returned arguments (apart from the error) are stored as possible values for other plugins to grab.

Plugins do not need to be specified in any order. They will be automatically reordered and loaded.

## Other

Any documentation missing from here will be added on request. Please feel free to file a bug.