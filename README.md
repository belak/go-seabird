# Seabird

[![Build Status](https://travis-ci.org/belak/go-seabird.svg?branch=master)](https://travis-ci.org/belak/go-seabird)

## Requirements

* Go 1.6
* gcc
* sqlite3 or postgresql

```
apt-get install golang gcc sqlite3
```

## Configuring

A sample config file is provided [here](./config.toml). Note that this
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

CREATE TABLE IF NOT EXISTS phrases (
    id SERIAL PRIMARY KEY,
    key VARCHAR(512) NOT NULL,
    value VARCHAR(512) DEFAULT '',
    submitter VARCHAR(512) NOT NULL,
    deleted BOOLEAN DEFAULT false
);

CREATE TABLE IF NOT EXISTS reminders (
    id SERIAL PRIMARY KEY,
    target VARCHAR(100) NOT NULL,
    target_type VARCHAR(10) NOT NULL,
    content VARCHAR(512) NOT NULL,
    reminder_time TIMESTAMP NOT NULL
);
```

This should work for both sqlite or postgres.
