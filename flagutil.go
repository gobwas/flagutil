package flagutil

import (
	"flag"
	"fmt"
	"os"

	"github.com/gobwas/flagutil/parse"
)

var SetSeparator = "."

type Parser interface {
	Parse(parse.FlagSet) error
}

type ParseOption func(*config)

func WithParser(p Parser) ParseOption {
	return func(c *config) {
		c.parsers = append(c.parsers, p)
	}
}

func WithIgnoreUndefined() ParseOption {
	return func(c *config) {
		c.ignoreUndefined = true
	}
}

type config struct {
	parsers         []Parser
	ignoreUndefined bool
}

func Parse(flags *flag.FlagSet, opts ...ParseOption) (err error) {
	var c config
	for _, opt := range opts {
		opt(&c)
	}
	fs := parse.NewFlagSet(flags,
		parse.WithIgnoreUndefined(c.ignoreUndefined),
	)
	for _, p := range c.parsers {
		parse.NextLevel(fs)
		if err = p.Parse(fs); err != nil {
			if err == flag.ErrHelp {
				printUsage(flags)
			}
			switch flags.ErrorHandling() {
			case flag.ContinueOnError:
				return err
			case flag.ExitOnError:
				fmt.Fprintf(flags.Output(), "flagutil: parse error: %v\n", err)
				os.Exit(2)
			case flag.PanicOnError:
				panic(err)
			}
		}
	}
	return nil
}

func printUsage(f *flag.FlagSet) {
	if f.Usage != nil {
		f.Usage()
		return
	}
	if name := f.Name(); name == "" {
		fmt.Fprintf(f.Output(), "Usage:\n")
	} else {
		fmt.Fprintf(f.Output(), "Usage of %s:\n", name)
	}
	f.PrintDefaults()
}

// Subset registers new flag subset with given prefix within given flag
// superset. It calls setup function to let caller register needed flags within
// created subset.
func Subset(super *flag.FlagSet, prefix string, setup func(sub *flag.FlagSet)) (err error) {
	sub := flag.NewFlagSet(prefix, 0)
	setup(sub)
	sub.VisitAll(func(f *flag.Flag) {
		name := prefix + "." + f.Name
		if super.Lookup(name) != nil && err == nil {
			err = fmt.Errorf(
				"flag %q already exists in a super set",
				name,
			)
			return
		}
		super.Var(f.Value, name, f.Usage)
	})
	return
}
