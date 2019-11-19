# Developing a Seabird URL Plugin

URL plugins are a special kind of Seabird plugin. Instead of defining commands, URL plugins watch for messages containing URLs that match certain domains. When such a message is seen, the plugin will execute a registered hook that will end with a message to a channel.

An example use case for this type of plugin is a hook that reads a GitHub repository URL and then sends a short message containing the repository's name, author, and other metadata.

You should put your URL plugins under `plugins/url/`.

## Defining a New URL Plugin

Here's a simple URL plugin to get you started:

```go
package url

import (
    seabird "github.com/belak/go-seabird"
)

func init() {
    seabird.RegisterPlugin("url/my_cool_url", newMyCoolUrlProvider)
}

func newMyCoolUrlProvider(b *seabird.Bot) error {
    err := b.EnsurePlugin("url")
    if err != nil {
        return err
    }

    urlPlugin := CtxPlugin(b.Context())
    urlPlugin.RegisterProvider("my.cool.url", readUrl)
}

func readUrl(r *seabird.Request, url *url.URL) bool {
    r.Reply("Your message contained a link to my.cool.url!")
}
```

This plugin adds a single URL hook that sends a message saying `Your message contained a link to my.cool.url!` any time a URL linking to the `my.cool.url` domain is sent to a channel where Seabird is present.

Note that you must add the plugin's name (`url/my_cool_url` from `init()`) to your bot's configuration under the `plugins` list for the new hook to be active.

## Plugin Configuration

To configure your URL plugin, you can create an object to wrap your configuration:

```go
package url

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
