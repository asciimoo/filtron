# Filtron

Reverse HTTP proxy to filter requests by different rules.
Can be used between production webserver and the application server to prevent abuse of the application backend.


## Installation and setup

```
$ go get github.com/asciimoo/filtron
$ "$GOPATH/bin/filtron" --help
```


## Rules

A rule has two required attributes:

 - `limit` integer - Defines how many matching requests allowed to access the application within `interval` seconds
 - `interval` integer - Time range in seconds to reset rule numbers

A rule also has to contain at least one of the following attributes:

 - `filters` list of selectors
 - `aggregations` list of selectors (if `filters` specified it activates only in case of the filter matches)


JSON representation of a rule:

```JSON
{
    "interval": 60,
    "limit": 10,
    "filters": ["GET:q", "Header:User-Agent=^curl"]
}
```
Explanation: Allow only 10 requests a minute where `q` represented as GET parameter and the user agent header starts with `curl`


### Filters

If all the selectors found, it increments a counter. Rule blocks the request if counter reaches `limit`


### Aggregation

Counts the values returned by selectors. Rule blocks the request if any value's number reaches `limit`


## Selectors

Selection of a request's different parts can be achieved using selector expressions.

Selectors are strings that can match any attribute of a HTTP request with the following syntax:

```
[!]RequestAttribute[:SubAttribute][=Regex]
```

 - `!` can negate the selector
 - `RequestAttribute` (required) selects specific part of a request - possible values:
    - Single value
      - `IP`
      - `Host`
      - `Path`
    - Multiple values
      - `GET`
      - `POST`
      - `Param` - it is an alias for both `GET` and `POST`
      - `Cookie`
      - `Header`
 - `SubAttribute` if `RequestAttribute` is not a single value, this can specify the inner attribute
 - `Regex` regular expression to filter the selected attributes value


### Examples

`IP` returns the client's IP address

`GET:x` returns the `x` GET parameter if exists

`!Header:Accept-Language` returns true if there is no `Accept-Language` HTTP header

`Path=^/(x|y)$` matches if the path is `/x` or `/y`


## API

Filtron can be configured through its REST API which listens on `127.0.0.1:4005` by default.

Currently it only reloads the rule file if `/reload_rules` called


## Bugs

Bugs or suggestions? Visit the [issue tracker](https://github.com/asciimoo/exrex/issues).
