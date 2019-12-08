# flagutil

[![GoDoc][godoc-image]][godoc-url]
[![Travis][travis-image]][travis-url]

> A library to populate [`flag.FlagSet`][flagset] from various sources.

# Features

- Uses standard `flag` package
- Structured configuration
- Reads values from multiple sources
- Ability to write your own parser

# Why

Defining your program parameters and reading their values should be as simple
as possible. There is no reason to use some library or even monstrous framework
instead of standard [flag][flag] package.

There is [Configuration in Go][article] article, which describes in detail the
reasons of creating this library.

# Usage

```go
package main

import (
	"flag"

	"github.com/gobwas/flagutil"
	"github.com/gobwas/flagutil/parse/pargs"
	"github.com/gobwas/flagutil/parse/file/json"
)

func main() {
	flags := flag.NewFlagSet("my-app", flag.ExitOnError)
	
	// Define top level parameters.
	var port int
	flag.IntVar(&port,
		"port", "port",
		"port to bind to",
	)

	// Now define parameters for some third-party library.
	var endpoint string
	flagutil.Subset(flags, "database", func(sub *flag.FlagSet) {
		sub.StringVar(&endpoint,
			"endpoint", "localhost",
			"database endpoint to connect to"
		)
	})
	
	// This flag is required be the file.Parser below.
	flags.String(
		"config", "/etc/app/config.json", 
		"path to configuration file",
	)

	flagutil.Parse(flags,
		// First, use posix options syntax instead of `flag` – just to
		// illustrate that it is possible.
		flagutil.WithParser(&pargs.Parser{
			Args: os.Args[1:],
		}),	

		// Then lookup for "config" flag value and try to parse its value as a
		// json configuration file.
		flagutil.WithParser(&file.Parser{
			PathFlag: "config",
			Syntax:   &json.Syntax{},
		}),
	)

	// Work with received values.
}
```

The configuration file may look as follows:

```json
{
  "port": 4050,
  
  "database": {
    "endpoint": "localhost:5432",
  }
}
```

And, if you want to override, say, database endpoint, you can execute your
program as follows:

```bash
$ app --config config.json --database.endpoint 4055
```

# Available parsers

Note that it is very easy to implement your own [Parser][parser].

At the moment these parsers are already implemented:
- [Flag][flag-syntax] syntax arguments parser
- [Posix][posix] program arguments syntax parser
- File parsers:
  - json
  - yaml
  - toml

# Conventions and limitations

Any structure from parsed configuration is converted into a pairs of a flat key
and a value. Keys are flattened recursively until there is no such flag defined
within `flag.FlagSet`.

> Keys flattening happens just as two keys concatenation with `.` as a >
> delimiter.

There are three scenarios when the flag was found:

1) If value is a mapping or an object, then its key-value pairs are
   concatenated with `:` as a delimiter and are passed to the `flag.Value.Set()`
   in appropriate number of calls.

2) If value is an array, then its items are passed to the `flag.Value.Set()` in
   appropriate number of calls. 

3) In other way, `flag.Value.Set()` will be called once with value as is.

Suppose you have this json configuration:

```json
{
  "foo": {
    "bar": "1",
    "baz": "2"
  }
}
```

If you define `foo.bar` flag, you will receive `"1"` in a single call to its
`flag.Value.Set()` method. No surprise here. But if you define `foo` flag, then
its `flag.Value.Set()` will be called twice with `"bar:1"` and `"baz:2"`.

The same thing happens with slices:

```json
{
  "foo": [
    "bar",
    "baz"
  ]
}
```

Your `foo`'s `flag.Value.Set()` will be called twice with `"bar"` and `"baz"`.

This still allows you to use command line arguments to override or declare
parameter complex values:

```bash
$ app --slice 1 --slice 2 --slice 3 --map foo:bar --map bar:baz
```

# Misc

Creation of this library was greatly inspired by [peterburgon/ff][ff] – and I
wouldn't write `flagutil` if I didn't met some design disagreement with it.


[parser]:       https://godoc.org/github.com/gobwas/flagutil#Parser
[flag]:         https://golang.org/pkg/flag
[flagset]:      https://golang.org/pkg/flag#FlagSet
[flag-syntax]:  https://golang.org/pkg/flag/#hdr-Command_line_flag_syntax
[article]:      https://gbws.io/articles/configuration-in-go
[godoc-image]:  https://godoc.org/github.com/gobwas/flagutil?status.svg
[godoc-url]:    https://godoc.org/github.com/gobwas/flagutil
[travis-image]: https://travis-ci.org/gobwas/flagutil.svg?branch=master
[travis-url]:   https://travis-ci.org/gobwas/flagutil
[posix]:        https://www.gnu.org/software/libc/manual/html_node/Argument-Syntax.html
[ff]:           https://github.com/peterbourgon/ff
