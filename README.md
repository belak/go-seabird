# Seabird

[![Build Status](https://drone.coded.io/api/badges/belak/go-seabird/status.svg)](https://drone.coded.io/belak/go-seabird)

## Requirements

* go >= 1.7
* gcc
* [vgo](https://golang.org/x/vgo)

## Configuring

A sample config file is provided [here](./_extra/config.sample.toml). Note that
this config file only has values specified for plugins. Some may not be needed.

Config is pulled from the environment variable `SEABIRD_CONFIG`. Set with

```
export SEABIRD_CONFIG=$HOME/config.toml
```

# Running

```
SEABIRD_CONFIG=$HOME/config.toml vgo run cmd/seabird/main.go
```

# License

[MIT](LICENSE)
