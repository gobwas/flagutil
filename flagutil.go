package flagutil

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/gobwas/flagutil/parse"
)

var SetSeparator = "."

type Parser interface {
	Parse(context.Context, parse.FlagSet) error
}

type ParserFunc func(context.Context, parse.FlagSet) error

func (fn ParserFunc) Parse(ctx context.Context, fs parse.FlagSet) error {
	return fn(ctx, fs)
}

type Printer interface {
	Name(context.Context, parse.FlagSet) (func(*flag.Flag, func(string)), error)
}

type PrinterFunc func(context.Context, parse.FlagSet) (func(*flag.Flag, func(string)), error)

func (fn PrinterFunc) Name(ctx context.Context, fs parse.FlagSet) (func(*flag.Flag, func(string)), error) {
	return fn(ctx, fs)
}

type parser struct {
	Parser
	stash               func(*flag.Flag) bool
	ignoreUndefined     bool
	allowResetSpecified bool
}

type config struct {
	parsers          []*parser
	parserOptions    []ParserOption
	customUsage      bool
	unquoteUsageMode UnquoteUsageMode
}

func buildConfig(opts []ParseOption) config {
	c := config{
		unquoteUsageMode: UnquoteDefault,
	}
	for _, opt := range opts {
		opt.setupParseConfig(&c)
	}
	for _, opt := range c.parserOptions {
		for _, p := range c.parsers {
			opt.setupParserConfig(p)
		}
	}
	return c
}

func Parse(ctx context.Context, flags *flag.FlagSet, opts ...ParseOption) (err error) {
	c := buildConfig(opts)

	fs := parse.NewFlagSet(flags)
	for _, p := range c.parsers {
		parse.NextLevel(fs)
		parse.Stash(fs, p.stash)
		parse.IgnoreUndefined(fs, p.ignoreUndefined)
		parse.AllowResetSpecified(fs, p.allowResetSpecified)

		if err = p.Parse(ctx, fs); err != nil {
			if err == flag.ErrHelp {
				_ = printUsageMaybe(ctx, &c, flags)
			}
			if err != nil {
				err = fmt.Errorf("flagutil: parse error: %w", err)
			}
			switch flags.ErrorHandling() {
			case flag.ContinueOnError:
				return err
			case flag.ExitOnError:
				if err != flag.ErrHelp {
					fmt.Fprintf(flags.Output(), "%v\n", err)
				}
				os.Exit(2)
			case flag.PanicOnError:
				panic(err.Error())
			}
		}
	}
	return nil
}

// PrintDefaults prints parsers aware usage message to flags.Output().
func PrintDefaults(ctx context.Context, flags *flag.FlagSet, opts ...ParseOption) error {
	c := buildConfig(opts)
	return printDefaults(ctx, &c, flags)
}

func printUsageMaybe(ctx context.Context, c *config, flags *flag.FlagSet) error {
	if !c.customUsage && flags.Usage != nil {
		flags.Usage()
		return nil
	}
	if name := flags.Name(); name == "" {
		fmt.Fprintf(flags.Output(), "Usage:\n")
	} else {
		fmt.Fprintf(flags.Output(), "Usage of %s:\n", name)
	}
	return printDefaults(ctx, c, flags)
}

type UnquoteUsageMode uint8

const (
	UnquoteNothing UnquoteUsageMode = 1 << iota >> 1
	UnquoteQuoted
	UnquoteInferType
	UnquoteClean

	UnquoteDefault UnquoteUsageMode = UnquoteQuoted | UnquoteInferType
)

func (m UnquoteUsageMode) String() string {
	switch m {
	case UnquoteNothing:
		return "UnquoteNothing"
	case UnquoteQuoted:
		return "UnquoteQuoted"
	case UnquoteInferType:
		return "UnquoteInferType"
	case UnquoteClean:
		return "UnquoteClean"
	case UnquoteDefault:
		return "UnquoteDefault"
	default:
		return "<unknown>"
	}
}

func (m UnquoteUsageMode) has(x UnquoteUsageMode) bool {
	return m&x != 0
}

func printDefaults(ctx context.Context, c *config, flags *flag.FlagSet) (err error) {
	fs := parse.NewFlagSet(flags)

	var hasNameFunc bool
	nameFunc := make([]func(*flag.Flag, func(string)), len(c.parsers))
	for i := len(c.parsers) - 1; i >= 0; i-- {
		if p, ok := c.parsers[i].Parser.(Printer); ok {
			hasNameFunc = true
			nameFunc[i], err = p.Name(ctx, fs)
			if err != nil {
				return
			}
		}
	}

	var buf bytes.Buffer
	flags.VisitAll(func(f *flag.Flag) {
		n, _ := buf.WriteString("  ")
		for i := len(c.parsers) - 1; i >= 0; i-- {
			fn := nameFunc[i]
			if fn == nil {
				continue
			}
			if stash := c.parsers[i].stash; stash != nil && stash(f) {
				continue
			}
			fn(f, func(name string) {
				if buf.Len() > n {
					buf.WriteString(", ")
				}
				buf.WriteString(name)
			})
		}
		if buf.Len() == n {
			// No name has been given.
			// Two cases are possible: no Printer implementation among parsers;
			// or some parser intentionally filtered out this flag.
			if hasNameFunc {
				buf.Reset()
				return
			}
			buf.WriteString(f.Name)
		}
		name, usage := unquoteUsage(c.unquoteUsageMode, f)
		if len(name) > 0 {
			buf.WriteString("\n    \t")
			buf.WriteString(name)
		}
		value := defValue(f)
		buf.WriteString("\n    \t")
		if len(usage) > 0 {
			buf.WriteString(strings.ReplaceAll(usage, "\n", "\n    \t"))
			if len(value) > 0 {
				buf.WriteString(" (")
			}
		}
		if len(value) > 0 {
			buf.WriteString("default ")
			buf.WriteString(defValue(f))
			if len(usage) > 0 {
				buf.WriteString(")")
			}
		}

		buf.WriteByte('\n')
		buf.WriteByte('\n')
		buf.WriteTo(flags.Output())
	})

	return nil
}

func defValue(f *flag.Flag) string {
	var x interface{}
	g, ok := f.Value.(flag.Getter)
	if ok {
		x = g.Get()
	}
	if def := f.DefValue; def != "" {
		if _, ok := x.(string); ok {
			def = `"` + def + `"`
		}
		return def
	}
	v := reflect.ValueOf(x)
repeat:
	if !v.IsValid() {
		return ""
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		goto repeat
	}
	switch v.Kind() {
	case reflect.String:
		return `""`
	case
		reflect.Slice,
		reflect.Array:
		return `[]`
	case
		reflect.Struct,
		reflect.Map:
		return `{}`
	}
	return ""
}

// defValueStd returns default value as it does std lib.
// NOTE: this one is unused.
func defValueStd(f *flag.Flag) string {
	t := reflect.TypeOf(f.Value)
	var z reflect.Value
	if t.Kind() == reflect.Ptr {
		z = reflect.New(t.Elem())
	} else {
		z = reflect.Zero(t)
	}
	zero := z.Interface().(flag.Value).String()
	if v := f.DefValue; v != zero {
		return v
	}
	return ""
}

// unquoteUsage is the same as flag.UnquoteUsage() with exception that it does
// not infer type of the flag value.
func unquoteUsage(m UnquoteUsageMode, f *flag.Flag) (name, usage string) {
	if m == UnquoteNothing {
		return "", f.Usage
	}
	u := f.Usage
	i := strings.IndexByte(u, '`')
	if i == -1 {
		if m.has(UnquoteInferType) {
			return inferType(f), f.Usage
		}
		return "", u
	}
	j := strings.IndexByte(u[i+1:], '`')
	if j == -1 {
		if m.has(UnquoteInferType) {
			return inferType(f), f.Usage
		}
		return "", u
	}
	j += i + 1

	switch {
	case m.has(UnquoteQuoted):
		name = u[i+1 : j]
	case m.has(UnquoteInferType):
		name = inferType(f)
	}

	prefix := u[:i]
	suffix := u[j+1:]
	switch {
	case m.has(UnquoteClean):
		usage = "" +
			strings.TrimRight(prefix, " ") +
			" " +
			strings.TrimLeft(suffix, " ")

	case m.has(UnquoteQuoted):
		usage = prefix + name + suffix

	default:
		usage = f.Usage
	}

	return
}

func inferType(f *flag.Flag) string {
	if f.Value == nil {
		return "?"
	}
	if isBoolFlag(f) {
		return "bool"
	}

	var x interface{}
	if g, ok := f.Value.(flag.Getter); ok {
		x = g.Get()
	} else {
		x = f.Value
	}
	v := reflect.ValueOf(x)

repeat:
	if !v.IsValid() {
		return ""
	}
	switch v.Type() {
	case reflect.TypeOf(time.Duration(0)):
		return "duration"
	}
	switch v.Kind() {
	case
		reflect.Interface,
		reflect.Ptr:

		v = v.Elem()
		goto repeat

	case
		reflect.String:
		return "string"
	case
		reflect.Float32,
		reflect.Float64:
		return "float"
	case
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64:
		return "int"
	case
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:
		return "uint"
	case
		reflect.Slice,
		reflect.Array:
		return "list"
	case
		reflect.Map:
		return "object"
	}
	return ""
}

// Subset registers new flag subset with given prefix within given flag
// superset. It calls setup function to let caller register needed flags within
// created subset.
func Subset(super *flag.FlagSet, prefix string, setup func(sub *flag.FlagSet)) (err error) {
	sub := flag.NewFlagSet(prefix, 0)
	setup(sub)
	sub.VisitAll(func(f *flag.Flag) {
		name := prefix + "." + f.Name
		if super.Lookup(name) != nil {
			if err == nil {
				err = fmt.Errorf(
					"flag %q already exists in a super set",
					name,
				)
				// TODO: should we panic here if super has PanicOnError?
			}
			return
		}
		super.Var(f.Value, name, f.Usage)
	})
	return
}

func isBoolFlag(f *flag.Flag) bool {
	return isBoolValue(f.Value)
}
func isBoolValue(v flag.Value) bool {
	x, ok := v.(interface {
		IsBoolFlag() bool
	})
	return ok && x.IsBoolFlag()
}

type joinVar struct {
	flags []*flag.Flag
}

// MergeUsage specifies way of joining two different flag usage strings.
var MergeUsage = func(name string, usage0, usage1 string) string {
	return usage0 + " / " + usage1
}

// MergeInto merges new flagset into superset and resolves any name collisions.
// It calls setup function to let caller register needed flags within subset
// before they are merged into the superset.
//
// If name of the flag defined in the subset already present in a superset,
// then subset flag is merged into superset's one. That is, flag will remain in
// the superset, but setting its value will make both parameters filled with
// received value.
//
// Description of each flag (if differ) is joined by MergeUsage().
//
// Note that default values (and initial values of where flag.Value points to)
// are kept untouched and may differ if no value is set during parsing phase.
func MergeInto(super *flag.FlagSet, setup func(*flag.FlagSet)) {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	setup(fs)
	fs.VisitAll(func(next *flag.Flag) {
		prev := super.Lookup(next.Name)
		if prev == nil {
			super.Var(next.Value, next.Name, next.Usage)
			return
		}
		*prev = *CombineFlags(prev, next)
	})
}

// Copy defines all flags from src with dst.
// It panics on any flag name collision.
func Copy(dst, src *flag.FlagSet) {
	src.VisitAll(func(f *flag.Flag) {
		dst.Var(f.Value, f.Name, f.Usage)
	})
}

// CombineSets combines given sets into a third one.
// Every collided flags are combined into third one in a way that setting value
// to it sets value of both original flags.
func CombineSets(fs0, fs1 *flag.FlagSet) *flag.FlagSet {
	// TODO: join Name().
	super := flag.NewFlagSet("", flag.ContinueOnError)
	fs0.VisitAll(func(f0 *flag.Flag) {
		var v flag.Value
		f1 := fs1.Lookup(f0.Name)
		if f1 != nil {
			// Same flag exists in fs1 flag set.
			f0 = CombineFlags(f0, f1)
		}
		v = OverrideSet(f0.Value, func(value string) (err error) {
			err = fs0.Set(f0.Name, value)
			if err != nil {
				return
			}
			if f1 == nil {
				return
			}
			err = fs1.Set(f1.Name, value)
			if err != nil {
				return
			}
			return nil
		})
		super.Var(v, f0.Name, f0.Usage)
	})
	fs1.VisitAll(func(f1 *flag.Flag) {
		if super.Lookup(f1.Name) != nil {
			// Already combined.
			return
		}
		v := OverrideSet(f1.Value, func(value string) error {
			return fs1.Set(f1.Name, value)
		})
		super.Var(v, f1.Name, f1.Usage)
	})
	return super
}

// CombineFlags combines given flags into a third one. Setting value of
// returned flag will cause both given flags change their values as well.
// However, flag sets of both flags will not be aware that the flags were set.
//
// Description of each flag (if differ) is joined by MergeUsage().
func CombineFlags(f0, f1 *flag.Flag) *flag.Flag {
	if f0.Name != f1.Name {
		panic(fmt.Sprintf(
			"flagutil: can't combine flags with different names: %q vs %q",
			f0.Name, f1.Name,
		))
	}
	r := flag.Flag{
		Name:  f0.Name,
		Value: valuePair{f0.Value, f1.Value},
		Usage: mergeUsage(f0.Name, f0.Usage, f1.Usage),
	}
	// This is how flag.FlagSet() does it in its Var() method.
	r.DefValue = r.Value.String()
	return &r
}

// SetActual makes flag look like it has been set within flag set.
// If flag set doesn't has flag with given SetActual() does nothing.
// Original value of found flag remains untouched, so it is safe to use with
// flags that accumulate values of multiple Set() calls.
func SetActual(fs *flag.FlagSet, name string) {
	f := fs.Lookup(name)
	if f == nil {
		return
	}
	orig := f.Value
	defer func() {
		f.Value = orig
	}()
	var didSet bool
	f.Value = value{
		doSet: func(s string) error {
			didSet = s == "dummy"
			return nil
		},
	}
	fs.Set(name, "dummy")
	if !didSet {
		panic("flagutil: make specified didn't work well")
	}
}

func mergeUsage(name, s0, s1 string) string {
	switch {
	case s0 == "":
		return s1
	case s1 == "":
		return s0
	case s0 == s1:
		return s0
	default:
		return MergeUsage(name, s0, s1)
	}
}
