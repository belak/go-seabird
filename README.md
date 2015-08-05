# seabird Plugins

The plugins are a set of plugins maintained by the authors of seabird
which are decent general use plugins.

## Requirements

 * Go 1.4
 * gcc
 * sqlite3 or postgresql

```
apt-get install golang gcc sqlite3
```

## Configuring

A sample config file is provided [here](./example/config.toml). Note that this
config file only has values specified for plugins. Some may not be needed.

Config is pulled from the environment variable `SEABIRD_CONFIG`. Set with

```
export SEABIRD_CONFIG=$HOME/config.toml
```

## Running

Once the config file is set, create the database with the
following schema:

```
CREATE TABLE IF NOT EXISTS karma (
	id SERIAL PRIMARY KEY,
	name VARCHAR(512) UNIQUE,
	score INTEGER
);

CREATE TABLE IF NOT EXISTS lastseen (
	name VARCHAR(512),
	channel VARCHAR(100),
	lastseen INTEGER,
	UNIQUE(name, channel)
);

CREATE TABLE IF NOT EXISTS nicks (
	nick VARCHAR(512),
	channel VARCHAR(100),
	flags VARCHAR(50),
	UNIQUE(nick, channel)
);

CREATE TABLE IF NOT EXISTS forecast_location (
	nick VARCHAR(512),
	address VARCHAR(200),
	lat FLOAT,
	lon FLOAT,
	UNIQUE(nick)
);
```

This should work for both sqlite or postgres.
