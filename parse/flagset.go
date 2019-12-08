package parse

import "flag"

type FlagSet struct {
	dest            *flag.FlagSet
	ignoreUndefined bool
	provided        map[string]bool
}

type FlagSetOption func(*FlagSet)

func WithIgnoreUndefined(v bool) FlagSetOption {
	return func(fs *FlagSet) {
		fs.ignoreUndefined = v
	}
}

func NewFlagSet(flags *flag.FlagSet, opts ...FlagSetOption) *FlagSet {
	fs := &FlagSet{
		dest:     flags,
		provided: make(map[string]bool),
	}
	for _, opt := range opts {
		opt(fs)
	}
	fs.update()
	return fs
}

func (fs *FlagSet) Set(name, value string) error {
	if fs.provided[name] {
		return nil
	}
	defined := fs.dest.Lookup(name) != nil
	if !defined && name != "help" && name != "h" && fs.ignoreUndefined {
		return nil
	}
	return fs.dest.Set(name, value)
}

func (fs *FlagSet) ParseLevel(fn func()) {
	fn()
	fs.update()
}

func (fs *FlagSet) update() {
	fs.dest.Visit(func(f *flag.Flag) {
		fs.provided[f.Name] = true
	})
}

func (fs *FlagSet) VisitAll(fn func(*flag.Flag)) {
	fs.dest.VisitAll(func(f *flag.Flag) {
		fn(fs.clone(f))
	})
}

func (fs *FlagSet) Lookup(name string) *flag.Flag {
	f := fs.dest.Lookup(name)
	if f != nil {
		f = fs.clone(f)
	}
	return f
}

func (fs *FlagSet) Has(name string) bool {
	return fs.Lookup(name) != nil
}

func (fs *FlagSet) clone(f *flag.Flag) *flag.Flag {
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
	fs   *FlagSet
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
