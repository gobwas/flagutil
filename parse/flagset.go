package parse

import "flag"

type FlagGetter interface {
	Lookup(name string) *flag.Flag
	VisitAll(func(*flag.Flag))
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
	fs.(*flagSet).update()
}

type flagSet struct {
	dest            *flag.FlagSet
	ignoreUndefined bool
	provided        map[string]bool
}

func NewFlagSet(flags *flag.FlagSet, opts ...FlagSetOption) FlagSet {
	fs := &flagSet{
		dest:     flags,
		provided: make(map[string]bool),
	}
	for _, opt := range opts {
		opt(fs)
	}
	fs.update()
	return fs
}

func (fs *flagSet) Set(name, value string) error {
	if fs.provided[name] {
		return nil
	}
	defined := fs.dest.Lookup(name) != nil
	if !defined && name != "help" && name != "h" && fs.ignoreUndefined {
		return nil
	}
	return fs.dest.Set(name, value)
}

func (fs *flagSet) update() {
	fs.dest.Visit(func(f *flag.Flag) {
		fs.provided[f.Name] = true
	})
}

func (fs *flagSet) VisitAll(fn func(*flag.Flag)) {
	fs.dest.VisitAll(func(f *flag.Flag) {
		fn(fs.clone(f))
	})
}

func (fs *flagSet) Lookup(name string) *flag.Flag {
	f := fs.dest.Lookup(name)
	if f != nil {
		f = fs.clone(f)
	}
	return f
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
