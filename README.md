# seabird

seabird is a golang library written as a wrapper around [belak/irc](https://github.com/belak/irc) to make making IRC bots more convenient. Note that currently there is no stability guarantee and interfaces may change at any time.

## Building

Once you have go installed and configured, run the following:

```
go get github.com/belak/seabird
```

This will build the seabird binary and place it in your `$GOPATH/src`.

## Configuring

A sample config file is provided [here](./config.yml)

By default seabird will look for a config file in `/etc/seabird/` and `$HOME/.config/seabird/` with a base name of `seabird` and an extension of `yml`, `json`, or `toml`.

## Options

Command line options are as follows:

```
--config=
Specify an alternate config file location
```

## Usage

See [here](./bot)

## Plugins

TODO: Add plugin documentation

See [here](./bot) for plugin structure.

## Muxes

Muxes are the building blocks of seabird. Three muxes are provided on top of belak/irc's BasicMux.

* CommandMux - Separates PRIVMSG events out into commands with a configurable prefix.
* CTCPMux - Separates CTCP events out of PRIVMSG events.
* MentionMux - Tracks the bot's current name and filters PRIVMSG events which start with the bot's username.