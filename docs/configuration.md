# Seabird Configuration

Seabird has a single configuration file that combines general bot configuration options with plugin-specific options.

A sample config file is provided [here](../_extra/config.sample.toml). Note that
this config file only has values specified for the core.

**How do I enable a plugin?**

You can use the `plugins` configuration key to enable plugins:

```
$ cat _extra/config.sample.toml
...
plugins = [
  "db",
  "chance",
]
```

In this example the `db` and `chance` plugins are enabled.

**How can I enable a collection of similarly-named plugins?**

Similar to the previous point, you can use the `plugins` configuration key to enable a group of plugins using [glob syntax](https://github.com/gobwas/glob):

```
$ cat _extra/config.sample.toml
...
plugins = [
  "db",
  "url/*",
]
```

In this example the `db` is enabled, as well as all plugins whose names start with `"url/"`.

**What configuration options exist for Seabird?**

Configuration for the underlying [irc](gopkg.in/irc.v3) connection (see [irc.ClientConfig](https://godoc.org/gopkg.in/irc.v3#ClientConfig) for more information):

```
# User info
nick = "HelloWorld"
user = "seabird"
name = "seabird"
pass = "qwertyasdf"

pingfrequency = "10s"
pingtimeout = "10s"

sendlimit = "0"
sendburst = 4
```

Network connection information, used with either [net](https://golang.org/pkg/net/) or [tls](https://golang.org/pkg/crypto/tls/) depending on whether or not TLS is used:

```
# Combination of host and port to connect to
host = "chat.freenode.net:6697"

# Connect with TLS
tls = true

# From the Go docs:
#
# [tlsnoverify] controls whether a client verifies the
# server's certificate chain and host name.
# If [tlsnoverify] is true, TLS accepts any certificate
# presented by the server and any host name in that certificate.
# In this mode, TLS is susceptible to man-in-the-middle attacks.
# This should be used only for testing.
tlsnoverify false

# File paths for the X509 keypair to use when connecting with TLS
tlscert     "/path/to/certfile"
tlskey      "/path/to/keyfile"
```

IRC commands for the bot to send upon connecting:

```
cmds = [
  "JOIN #my-channel",
]
```

Command prefix for the bot, e.g. setting `prefix = "~"` would mean that you'd call a command named `foo` with a message like `~foo`.

```
prefix = "!"
```

As detailed above, `plugins` controls which plugins are enabled in the bot.

```
plugins = [
  "db",
  "runescape",
  "noaa",
]
```

`loglevel` controls the bot's log level. See [this](https://github.com/sirupsen/logrus/blob/master/logrus.go#L25) for supported levels. Note: `debug` has been deprecated. Don't use it.

```
loglevel = "info"
```

[documentation index](./README.md)
