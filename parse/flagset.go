package parse

import (
	"flag"
	"fmt"
)

type FlagGetter interface {
	Lookup(name string) *flag.Flag
	VisitAll(func(*flag.Flag))
	VisitUnspecified(func(*flag.Flag))
}

type FlagSetter interface {
	Set(name, value string) error
}

type FlagSet interface {
	FlagGetter
	FlagSetter
}

type FlagSetOption func(*flagSet)

func WithIgnoreUndefined(v bool) FlagSetOption {
	return func(fs *flagSet) {
		fs.ignoreUndefined = v
	}
}

func NextLevel(fs FlagSet) {
	fset := fs.(*flagSet)
	fset.aliases = nil
	fset.names = nil
	fset.update()
}

// Name makes fs present every flag to its parser under the aliases fn
// returns for the flag's name: the flag is visited once per alias, found
// and set by any of them, and not at all when fn returns none. A nil fn
// presents every flag under its own name. Name panics when fn gives two
// flags one alias.
func Name(fs FlagSet, fn func(name string) []string) {
	fset := fs.(*flagSet)
	fset.aliases = fn
	fset.names = nil
	if fn == nil {
		return
	}
	fset.names = make(map[string]string)
	fset.dest.VisitAll(func(f *flag.Flag) {
		for _, alias := range fn(f.Name) {
			if alias == "" {
				continue
			}
			if prev, has := fset.names[alias]; has {
				panic(fmt.Sprintf(
					"parse: flags %q and %q are both named %q",
					prev, f.Name, alias,
				))
			}
			fset.names[alias] = f.Name
		}
	})
}

func IgnoreUndefined(fs FlagSet, ignore bool) {
	fset := fs.(*flagSet)
	fset.ignoreUndefined = ignore
}

func AllowResetSpecified(fs FlagSet, allow bool) {
	fset := fs.(*flagSet)
	fset.allowResetSpecified = allow
}

type flagSet struct {
	dest                *flag.FlagSet
	ignoreUndefined     bool
	allowResetSpecified bool
	specified           map[string]bool

	// aliases maps a flag's name to the aliases its parser sees it by;
	// nil is the identity. names maps each alias back, built by Name; an
	// alias absent from it is no flag of the parser's.
	aliases func(name string) []string
	names   map[string]string
}

func NewFlagSet(flags *flag.FlagSet, opts ...FlagSetOption) FlagSet {
	fs := &flagSet{
		dest:      flags,
		specified: make(map[string]bool),
	}
	for _, opt := range opts {
		opt(fs)
	}
	fs.update()
	return fs
}

func (fs *flagSet) Set(alias, value string) error {
	name, defined := fs.resolve(alias)
	if defined && fs.specified[name] && !fs.allowResetSpecified {
		return nil
	}
	if !defined && fs.ignoreUndefined {
		return nil
	}
	if !defined {
		return &Error{
			Name:      alias,
			Value:     value,
			Undefined: true,
		}
	}
	if err := fs.dest.Set(name, value); err != nil {
		return &Error{
			Name:  alias,
			Value: value,
			Err:   err,
		}
	}
	return nil
}

// resolve returns the flag's name for an alias its parser uses, and
// whether the parser has such a flag.
func (fs *flagSet) resolve(alias string) (name string, ok bool) {
	if fs.aliases == nil {
		return alias, fs.dest.Lookup(alias) != nil
	}
	name, ok = fs.names[alias]
	return name, ok
}

// aliasesOf returns the aliases the parser sees f by, none when f is no
// flag of the parser's.
func (fs *flagSet) aliasesOf(f *flag.Flag) []string {
	if fs.aliases == nil {
		return []string{f.Name}
	}
	var aliases []string
	for _, alias := range fs.aliases(f.Name) {
		if alias != "" {
			aliases = append(aliases, alias)
		}
	}
	return aliases
}

func (fs *flagSet) update() {
	fs.dest.Visit(func(f *flag.Flag) {
		fs.specified[f.Name] = true
	})
}

func (fs *flagSet) VisitUnspecified(fn func(*flag.Flag)) {
	fs.dest.VisitAll(func(f *flag.Flag) {
		if fs.specified[f.Name] {
			return
		}
		for _, alias := range fs.aliasesOf(f) {
			fn(fs.clone(f, alias))
		}
	})
}

func (fs *flagSet) VisitAll(fn func(*flag.Flag)) {
	fs.dest.VisitAll(func(f *flag.Flag) {
		for _, alias := range fs.aliasesOf(f) {
			fn(fs.clone(f, alias))
		}
	})
}

func (fs *flagSet) Lookup(alias string) *flag.Flag {
	name, ok := fs.resolve(alias)
	if !ok {
		return nil
	}
	return fs.clone(fs.dest.Lookup(name), alias)
}

// clone copies f under one of the aliases the parser sees it by, with a
// Value whose Set goes through fs by that alias.
func (fs *flagSet) clone(f *flag.Flag, alias string) *flag.Flag {
	cp := *f
	cp.Name = alias
	cp.Value = value{
		Value: f.Value,
		fs:    fs,
		alias: alias,
	}
	return &cp
}

type value struct {
	flag.Value
	fs    *flagSet
	alias string
}

func (v value) Set(s string) error {
	return v.fs.Set(v.alias, s)
}

func (v value) IsBoolFlag() bool {
	x, ok := v.Value.(interface {
		IsBoolFlag() bool
	})
	return ok && x.IsBoolFlag()
}
