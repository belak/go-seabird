# Developing a Seabird Plugin

Seabird plugins require a bit of boilerplate, but should be reasonably simple to create.

Unless you're creating a [URL plugin](./url_plugin_development.md), you should put your plugins under `plugins/extra/`.

## Defining a New Plugin

Here's a simple plugin to get you started:

```go
package extra

import (
    seabird "github.com/belak/go-seabird"
)

func init() {
    seabird.RegisterPlugin("my_cool_plugin", newMyCoolPlugin)
}

func newMyCoolPlugin(b *seabird.Bot) error {
    cm := b.CommandMux()

    cm.Event("my_command", commandCallback, &seabird.HelpInfo{
        Description: "This command does something.",
    })

    return nil
}

func commandCallback(r *seabird.Request) {
    r.MentionReply("You ran my_command!")
}
```

This plugin adds a single command called `my_command` that will reply to a user when the command is called.

## Registering Callbacks

You've already seen one way to register bot callbacks in `CommandMux.Event`. There are a few different ways to register message callbacks in a plugin:

### `BasicMux`

`BasicMux().Event`: This will register a callback that will be called when Seabird sees specific raw IRC commands like `JOIN`, `PART`, and `KICK`.

### `CommandMux`

`CommandMux().Event`: This will register a callback that will be called for a specific command, either in a channel or in a private query. Only messages beginning with the bot's [configured command prefix](configuration.md) and the registered command (e.g. `~help`) will cause the callback to fire.

`CommandMux().Channel`: This will register a callback that will be called for a specific command only in a channel and not in a private query. Only messages beginning with the bot's [configured command prefix](configuration.md) and the registered command (e.g. `~help`) will cause the callback to fire.

`CommandMux().Private`: This will register a callback that will be called for a specific command only in a private query and not in a channel. Only messages beginning with the bot's [configured command prefix](configuration.md) and the registered command (e.g. `~help`) will cause the callback to fire.

### `MentionMux`

`MentionMux().Event`: This will register a callback that will be called for every message that a Seabird bot sees. This is useful for parsing specific, common parts of messages like URLs.

## Writing Messages

You may send messages to a channel in a number of ways. The following are three common ways to do it.

`Request().Reply`: This will simply send a message to the channel or private query that the source request came from.
`Request().MentionReply`: This will send a message prefixed with the issuing user's nick to the channel or private query that the source request came from.
`Request().PrivateReply`: This will open a private query with the user that issued the request and send the reply there.

## Depending on Other Plugins

You can depend on other plugins with the `Bot().EnsurePlugin` method.
```go
package extra

import (
    seabird "github.com/belak/go-seabird"
)

func init() {
    seabird.RegisterPlugin("my_cool_plugin", newMyCoolPlugin)
}

func newMyCoolPlugin(b *seabird.Bot) error {
    // This will require that the plugin "some_other_plugin"
    // is loaded.
    err := b.EnsurePlugin("some_other_plugin")
    if err != nil {
        return err
    }

    cm := b.CommandMux()

    cm.Event("my_command", commandCallback, &seabird.HelpInfo{
        Description: "This command does something.",
    })

    return nil
}

func commandCallback(r *seabird.Request) {
    r.MentionReply("You ran my_command!")
}
```

## Plugin Configuration

To configure your plugin, you can create an object to wrap your configuration:

```go
package extra

import (
    seabird "github.com/belak/go-seabird"
)

func init() {
    seabird.RegisterPlugin("url/my_cool_url", newMyCoolUrlProvider)
}

// myCoolUrlConfig defines configuration for the URL plugin
type myCoolUrlConfig struct {
    firstConfigValue  string
    secondConfigValue bool
}

// myCoolUrlProvider will store loaded configuration options
type myCoolUrlProvider struct {
    config *myCoolUrlConfig
}

func newMyCoolUrlProvider(b *seabird.Bot) error {
    err := b.EnsurePlugin("url")
    if err != nil {
        return err
    }

    p := &myCoolUrlProvider{}

    c := &myCoolUrlConfig{}
    if err := b.COnfig("my_cool_url", c); err != nil {
        return err
    }

    p.config = c

    urlPlugin := CtxPlugin(b.Context())
    urlPlugin.RegisterProvider("my.cool.url", readUrl)
}

func (p *myCoolUrlProvider) readUrl(r *seabird.Request, url *url.URL) bool {
    // You can now access configuration values with:
    // p.config.firstConfigValue

    r.Reply("Your message contained a link to my.cool.url!")
}
```

[documentation index](./README.md)
