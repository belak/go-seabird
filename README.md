# seabird

seabird is a golang library written as a wrapper around [belak/irc](https://github.com/belak/irc) to make making IRC bots more convenient. Note that currently there is no stability guarantee and interfaces may change at any time.

## Requirements

 * Go 1.4
 * Mercurial
 * gcc
 * sqlite3

```
apt-get install golang mercurial gcc sqlite3
```

## Building

Once you have go installed, set your GOPATH. For example

```
mkdir $HOME/go
export GOPATH=$HOME/go

export PATH=$PATH:$GOPATH/bin
```

run the following to download and build seabird:

```
go get github.com/belak/seabird
```

This will build the seabird binary and place it in your `$GOPATH/bin`.

## Configuring

A sample config file is provided [here](./config.toml)

Config is pulled from the environment variable SEABIRD_CONFIG. Set with

```
export SEABIRD_CONFIG=$HOME/config.toml
```

## Options

Command line options are as follows:

```
--config=
Specify an alternate config file location
```

## Running

Once the config file is set, create the sqlite database by running

```
cat $GOPATH/src/github.com/belak/seabird/schema.sql | sqlite3 dev.db
```

Start the bot by simply runnning

```
seabird
```
Note you can append `&` to the end of the seabird command to fork it to the background.

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
