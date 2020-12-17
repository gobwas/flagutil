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
	fset.stash = nil
	fset.update()
}

func Stash(fs FlagSet, fn func(*flag.Flag) bool) {
	fset := fs.(*flagSet)
	fset.stash = fn
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
	stash               func(*flag.Flag) bool
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

func (fs *flagSet) Set(name, value string) error {
	if fs.specified[name] && !fs.allowResetSpecified {
		return nil
	}
	f := fs.dest.Lookup(name)
	if f != nil && fs.stashed(f) {
		f = nil
	}
	defined := f != nil
	if !defined && fs.ignoreUndefined {
		return nil
	}
	if !defined {
		return fmt.Errorf("flag provided but not defined: %q", name)
	}
	err := fs.dest.Set(name, value)
	if err != nil {
		err = fmt.Errorf("set %q: %w", name, err)
	}
	return err
}

func (fs *flagSet) stashed(f *flag.Flag) bool {
	stash := fs.stash
	return stash != nil && stash(f)
}

func (fs *flagSet) update() {
	fs.dest.Visit(func(f *flag.Flag) {
		fs.specified[f.Name] = true
	})
}

func (fs *flagSet) VisitUnspecified(fn func(*flag.Flag)) {
	fs.dest.VisitAll(func(f *flag.Flag) {
		if !fs.specified[f.Name] && !fs.stashed(f) {
			fn(fs.clone(f))
		}
	})
}

func (fs *flagSet) VisitAll(fn func(*flag.Flag)) {
	fs.dest.VisitAll(func(f *flag.Flag) {
		if !fs.stashed(f) {
			fn(fs.clone(f))
		}
	})
}

func (fs *flagSet) Lookup(name string) *flag.Flag {
	f := fs.dest.Lookup(name)
	if f == nil || fs.stashed(f) {
		return nil
	}
	return fs.clone(f)
}

func (fs *flagSet) clone(f *flag.Flag) *flag.Flag {
	cp := *f
	cp.Value = value{
		Value: f.Value,
		fs:    fs,
		name:  f.Name,
	}
	return &cp
}

type value struct {
	flag.Value
	fs   *flagSet
	name string
}

func (v value) Set(s string) error {
	return v.fs.Set(v.name, s)
}

func (v value) IsBoolFlag() bool {
	x, ok := v.Value.(interface {
		IsBoolFlag() bool
	})
	return ok && x.IsBoolFlag()
}
