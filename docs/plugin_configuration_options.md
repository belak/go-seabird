# Plugin Configuration Options

Each of these is listed by its plugin name. Configuration for each plugin should follow its section name as listed under its heading.

## `bulkcnam`

This plugin is not currently working.

## `db`

Section name is `[db]`.

**Required**: `driver` specifies the database you're using. See [this](http://gobook.io/read/gitea.com/xorm/manual-en-US/) for supported databases.

```
driver = "sqlite3"
```

**Required**: `datasource` specifies connection instructions for the database you're using. This is specific to your database.

```
datasource = "dev.db"
```

_Optional_: `tableprefix` allows you to optionally prefix plugin-generated table names with a string.
```
tableprefix = "seabird_"
```

## `forecast`

Section name is `[forecast]`.

**Required:** `key` is a key to [Dark Sky's API](https://darksky.net/dev).

```
key = "abc123"
```

**Required:** `mapskey` is a key to the [Google Maps API](https://developers.google.com/maps/documentation).

```
mapskey = "def456"
```

## `issues`

Section name is `[issues]`.

**Required:** `token` is an access token for [GitHub's API](https://developer.github.com/v3/).

```
token = "abc123"
```

## `net_tools`

Section name is `[net_tools]`.

**Required:** `key` is a key for the [Pastebin API](https://pastebin.com/api/?ref=public-apis).

```
key = "jkl012"
```

## `stock`

Section name is `[stock]`.

**Required:** `key` is a key for the [IEX Cloud API](https://iexcloud.io/docs/api/).

```
key = "ghi789"
```

## `tiny`

Section name is `[tiny]`.

**Required:** `key` is a key for [Google's URL shortener API](https://developers.google.com/url-shortener).

```
key = "def456"
```

## `url/github`

Section name is `[github]`.

**Required:** `token` is an access token for [GitHub's API](https://developer.github.com/v3/).

```
token = "abc123"
```

## `url/spotify`

Section name is `[spotify]`.

**Required:** `clientid` is a client ID for the [Spotify API](https://developer.spotify.com/documentation/web-api/).

```
clientid = "pqr678"
```

**Required:** `clientsecret` is a client secret for the [Spotify API](https://developer.spotify.com/documentation/web-api/).

```
clientsecret = "stu901"
```

## `url/twitter`

This plugin is not currently working.

## `url/youtube`

Section name is `[youtube]`.

**Required:** `key` is a key for the [YouTube API](https://developers.google.com/youtube/v3).

```
key = "mno345"
```

[general configuration](./configuration.md)
