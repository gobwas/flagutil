# flagutil

[![GoDoc][godoc-image]][godoc-url]
[![CI][ci-badge]][ci-url]

> A library to populate [`flag.FlagSet`][flagSet] from various sources.

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

# Available parsers

Note that it is very easy to implement your own [Parser][parser].

At the moment these parsers are already implemented:
- [Flag][flag-syntax] syntax arguments parser
- [Posix][posix] program arguments syntax parser
- Environment variables parser
- Prompt interactive parser
- File parsers:
  - json
  - yaml
  - toml

# Custom help message

It is possible to print custom help message which may include, for example,
names of the environment variables used by env parser. See the
`WithCustomUsage()` parse option.

Custom usage currently looks like this:

```
Usage of test:
  $TEST_FOO, --foo
        bool
        bool flag description (default false)

  $TEST_BAR, --bar
        int
        int flag description (default 42)
```

# Usage

A simple example could be like this:

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
	
	port := flag.Int(&port,
		"port", "port",
		"port to bind to",
	)

	// This flag will be required by the file.Parser below.
	_ = flags.String(
		"config", "/etc/app/config.json", 
		"path to configuration file",
	)

	flagutil.Parse(flags,
		// Use posix options syntax instead of `flag` – just to illustrate that
		// it is possible.
		flagutil.WithParser(&pargs.Parser{
			Args: os.Args[1:],
		}),	

		// Then lookup flag values among environment.
		flagutil.WithParser(&env.Parser{
			Prefix: "MY_APP_",
		}),

		// Finally lookup for "config" flag value and try to interpret its
		// value as a path to json configuration file.
		flagutil.WithParser(
			&file.Parser{
				Lookup: file.LookupFlag(flags, "config"),
				Syntax: new(json.Syntax),
			},
			// Don't allow to setup "config" flag from file.
			flagutil.WithStashName("config"),
		),
	)

	// Work with received values.
}
```

However, `flagutil` provides ability to define so called flag subsets:

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
	
	port := flag.Int(&port,
		"port", "port",
		"port to bind to",
	)

	// Define parameters for some third-party library.
	var endpoint string
	flagutil.Subset(flags, "database", func(sub *flag.FlagSet) {
		sub.StringVar(&endpoint,
			"endpoint", "localhost",
			"database endpoint to connect to"
		)
	})
	
	flagutil.Parse(flags,
		flagutil.WithParser(&pargs.Parser{
			Args: os.Args[1:],
		}),	
		flagutil.WithParser(
			&file.Parser{
				Lookup: file.PathLookup("/etc/my-app/config.json"),
				Syntax: new(json.Syntax),
			},
		),
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
$ app --database.endpoint 4055
```

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

> Note that for any type of values the `flag.Value.String()` method is never
> used to access the "real" value – only for defaults when printing help
> message. To provide "real" value implementations must satisfy `flag.Getter`
> interface.

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
[flagSet]:      https://golang.org/pkg/flag#FlagSet
[flag-syntax]:  https://golang.org/pkg/flag/#hdr-Command_line_flag_syntax
[article]:      https://gbws.io/articles/configuration-in-go
[godoc-image]:  https://godoc.org/github.com/gobwas/flagutil?status.svg
[godoc-url]:    https://godoc.org/github.com/gobwas/flagutil
[posix]:        https://www.gnu.org/software/libc/manual/html_node/Argument-Syntax.html
[ff]:           https://github.com/peterbourgon/ff
[ci-badge]:     https://github.com/gobwas/flagutil/workflows/CI/badge.svg
[ci-url]:       https://github.com/gobwas/flagutil/actions?query=workflow%3ACI
