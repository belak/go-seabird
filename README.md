# Seabird

[![Build Status](https://travis-ci.org/belak/go-seabird.svg?branch=master)](https://travis-ci.org/belak/go-seabird)

## Requirements

* go >= 1.12
* gcc

## Configuring

A sample config file is provided [here](./_extra/config.sample.toml). Note that
this config file only has values specified for plugins. Some may not be needed.

Config is pulled from the environment variable `SEABIRD_CONFIG`. Set with

```
export SEABIRD_CONFIG=$HOME/config.toml
```

# Running

```
SEABIRD_CONFIG=$HOME/config.toml go run cmd/seabird/main.go
```

# License

[MIT](LICENSE)
